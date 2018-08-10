package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gsmcwhirter/discord-bot-lib/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/discordapi/session"
	"github.com/gsmcwhirter/discord-bot-lib/snowflake"
	"github.com/gsmcwhirter/discord-bot-lib/util"
	"github.com/gsmcwhirter/go-util/parser"
	"github.com/pkg/errors"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

// ErrNoResponse TODOC
var ErrNoResponse = errors.New("no response")

type dependencies interface {
	TrialAPI() storage.TrialAPI
	GuildAPI() storage.GuildAPI
	BotSession() *session.Session
}

// Options TODOC
type Options struct {
	CmdIndicator string
}

// RootCommands holds the commands at the root level
type rootCommands struct {
	deps       dependencies
	versionStr string
}

func (c *rootCommands) version(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To:          cmdhandler.UserMentionString(msg.UserID()),
		Description: c.versionStr,
	}
	return r, nil
}

func (c *rootCommands) list(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.EmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), false)
	if err != nil {
		return r, err
	}
	defer util.CheckDefer(t.Rollback)

	g, gerr := c.deps.BotSession().Guild(msg.GuildID())

	trials := t.GetTrials()
	tNames := make([]string, 0, len(trials))
	for _, trial := range trials {
		if trial.GetState() != storage.TrialStateClosed {
			var tscID snowflake.Snowflake
			var ok bool
			if gerr == nil {
				tscID, ok = g.ChannelWithName(trial.GetSignupChannel())
			}

			if ok {
				tNames = append(tNames, fmt.Sprintf("%s (%s)", trial.GetName(), cmdhandler.ChannelMentionString(tscID)))
			} else {
				tNames = append(tNames, trial.GetName())
			}
		}
	}
	sort.Strings(tNames)

	r.Fields = []cmdhandler.EmbedField{
		{
			Name: "*Available Trials*",
			Val:  strings.Join(tNames, "\n"),
		},
	}

	return r, nil
}

func (c *rootCommands) show(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	trialName := strings.TrimSpace(msg.Contents())

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), false)
	if err != nil {
		return r, err
	}
	defer util.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	r2 := formatTrialDisplay(trial, true)
	r2.To = cmdhandler.UserMentionString(msg.UserID())

	return r2, nil
}

func (c *rootCommands) signup(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	argParts := strings.SplitN(strings.TrimSpace(msg.Contents()), " ", 2)
	if len(argParts) < 2 {
		return r, errors.New("missing role")
	}

	trialName, role := argParts[0], argParts[1]

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer util.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	if trial.GetState() != storage.TrialStateOpen {
		return r, errors.New("cannot sign up for a closed trial")
	}

	overflow, err := signupUser(trial, cmdhandler.UserMentionString(msg.UserID()), role)
	if err != nil {
		return r, err
	}

	if err = t.SaveTrial(trial); err != nil {
		return r, errors.Wrap(err, "could not save trial signup")
	}

	if err = t.Commit(); err != nil {
		return r, errors.Wrap(err, "could not save trial signup")
	}

	if overflow {
		r.Description = fmt.Sprintf("Signed up as OVERFLOW for %s in %s", role, trialName)
	} else {
		r.Description = fmt.Sprintf("Signed up for %s in %s", role, trialName)
	}

	return r, nil
}

func (c *rootCommands) withdraw(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	trialName := strings.TrimSpace(msg.Contents())

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer util.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	if trial.GetState() != storage.TrialStateOpen {
		return r, errors.New("cannot withdraw from a closed trial")
	}

	trial.RemoveSignup(cmdhandler.UserMentionString(msg.UserID()))

	if err = t.SaveTrial(trial); err != nil {
		return r, errors.Wrap(err, "could not save trial withdraw")
	}

	if err = t.Commit(); err != nil {
		return r, errors.Wrap(err, "could not save trial withdraw")
	}

	r.Description = fmt.Sprintf("Withdrew from %s", trialName)

	return r, nil
}

// CommandHandler TODOC
func CommandHandler(deps dependencies, versionStr string, opts Options) *cmdhandler.CommandHandler {
	p := parser.NewParser(parser.Options{
		CmdIndicator: opts.CmdIndicator,
	})
	rh := rootCommands{
		deps:       deps,
		versionStr: versionStr,
	}

	ch := cmdhandler.NewCommandHandler(p, cmdhandler.Options{})
	ch.SetHandler("version", cmdhandler.NewMessageHandler(rh.version))
	ch.SetHandler("list", cmdhandler.NewMessageHandler(rh.list))
	ch.SetHandler("show", cmdhandler.NewMessageHandler(rh.show))
	ch.SetHandler("signup", cmdhandler.NewMessageHandler(rh.signup))
	ch.SetHandler("su", cmdhandler.NewMessageHandler(rh.signup))
	ch.SetHandler("withdraw", cmdhandler.NewMessageHandler(rh.withdraw))
	ch.SetHandler("wd", cmdhandler.NewMessageHandler(rh.withdraw))

	return ch
}

type configDependencies interface {
	GuildAPI() storage.GuildAPI
}

// ConfigHandler TODOC
func ConfigHandler(deps configDependencies, versionStr string, opts Options) *cmdhandler.CommandHandler {
	p := parser.NewParser(parser.Options{
		CmdIndicator: opts.CmdIndicator,
	})

	ch := cmdhandler.NewCommandHandler(p, cmdhandler.Options{
		NoHelpOnUnknownCommands: true,
	})
	ch.SetHandler("config-su", ConfigCommandHandler(deps, fmt.Sprintf("%sconfig", opts.CmdIndicator)))
	// disable help for config
	ch.SetHandler("help", cmdhandler.NewMessageHandler(func(msg cmdhandler.Message) (cmdhandler.Response, error) {
		r := &cmdhandler.SimpleEmbedResponse{}
		return r, parser.ErrUnknownCommand
	}))

	return ch
}

type adminDependencies interface {
	GuildAPI() storage.GuildAPI
	TrialAPI() storage.TrialAPI
	BotSession() *session.Session
}

// AdminHandler TODOC
func AdminHandler(deps adminDependencies, versionStr string, opts Options) *cmdhandler.CommandHandler {
	p := parser.NewParser(parser.Options{
		CmdIndicator: opts.CmdIndicator,
	})

	ch := cmdhandler.NewCommandHandler(p, cmdhandler.Options{
		NoHelpOnUnknownCommands: true,
	})

	ch.SetHandler("admin", AdminCommandHandler(deps, fmt.Sprintf("%sadmin", opts.CmdIndicator)))
	// disable help for admin
	ch.SetHandler("help", cmdhandler.NewMessageHandler(func(msg cmdhandler.Message) (cmdhandler.Response, error) {
		r := &cmdhandler.SimpleEmbedResponse{}
		return r, parser.ErrUnknownCommand
	}))

	return ch
}
