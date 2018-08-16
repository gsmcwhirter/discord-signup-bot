package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gsmcwhirter/discord-bot-lib/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/discordapi/session"
	"github.com/gsmcwhirter/discord-bot-lib/snowflake"
	"github.com/gsmcwhirter/discord-bot-lib/util"
	"github.com/gsmcwhirter/go-util/parser"
	"github.com/pkg/errors"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/logging"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

// ErrNoResponse TODOC
var ErrNoResponse = errors.New("no response")

type dependencies interface {
	Logger() log.Logger
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
	deps dependencies
}

func (c *rootCommands) list(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.EmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling rootCommand", "command", "list")

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

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling rootCommand", "command", "show", "trial_name", trialName)

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), false)
	if err != nil {
		return r, err
	}
	defer util.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	if !isSignupChannel(msg, trial.GetSignupChannel(), c.deps.BotSession()) {
		_ = level.Info(logger).Log("message", "command not in signup channel", "signup_channel", trial.GetSignupChannel())
		return r, msghandler.ErrNoResponse
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

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling rootCommand", "command", "signup", "trial_and_role", argParts)

	if len(argParts) < 2 {
		return r, errors.New("missing role")
	}

	trialName, role := argParts[0], argParts[1]

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID().ToString())
	if err != nil {
		return r, err
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer util.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	if !isSignupChannel(msg, trial.GetSignupChannel(), c.deps.BotSession()) {
		_ = level.Info(logger).Log("message", "command not in signup channel", "signup_channel", trial.GetSignupChannel())
		return r, msghandler.ErrNoResponse
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

	var descStr string
	if overflow {
		_ = level.Info(logger).Log("message", "signed up", "overflow", true, "role", role, "trial_name", trialName)
		descStr = fmt.Sprintf("Signed up as OVERFLOW for %s in %s", role, trialName)
	} else {
		_ = level.Info(logger).Log("message", "signed up", "overflow", false, "role", role, "trial_name", trialName)
		descStr = fmt.Sprintf("Signed up for %s in %s", role, trialName)
	}

	if gsettings.ShowAfterSignup == "true" {
		_ = level.Debug(logger).Log("message", "auto-show after signup", "trial_name", trialName)

		r2 := formatTrialDisplay(trial, true)
		r2.To = cmdhandler.UserMentionString(msg.UserID())
		r2.Description = fmt.Sprintf("%s\n\n%s", descStr, r2.Description)
		return r2, nil
	}

	r.Description = descStr

	return r, nil
}

func (c *rootCommands) withdraw(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	trialName := strings.TrimSpace(msg.Contents())

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling rootCommand", "command", "withdraw", "trial_name", trialName)

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID().ToString())
	if err != nil {
		return r, err
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer util.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	if !isSignupChannel(msg, trial.GetSignupChannel(), c.deps.BotSession()) {
		_ = level.Info(logger).Log("message", "command not in signup channel", "signup_channel", trial.GetSignupChannel())
		return r, msghandler.ErrNoResponse
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

	_ = level.Info(logger).Log("message", "withdrew", "trial_name", trialName)
	descStr := fmt.Sprintf("Withdrew from %s", trialName)

	if gsettings.ShowAfterWithdraw == "true" {
		_ = level.Debug(logger).Log("message", "auto-show after withdraw", "trial_name", trialName)

		r2 := formatTrialDisplay(trial, true)
		r2.To = cmdhandler.UserMentionString(msg.UserID())
		r2.Description = fmt.Sprintf("%s\n\n%s", descStr, r2.Description)
		return r2, nil
	}

	r.Description = descStr

	return r, nil
}

// CommandHandler TODOC
func CommandHandler(deps dependencies, versionStr string, opts Options) (*cmdhandler.CommandHandler, error) {
	p := parser.NewParser(parser.Options{
		CmdIndicator: opts.CmdIndicator,
	})
	rh := rootCommands{
		deps: deps,
	}

	ch, err := cmdhandler.NewCommandHandler(p, cmdhandler.Options{
		NoHelpOnUnknownCommands: true,
	})
	if err != nil {
		return nil, err
	}

	ch.SetHandler("list", cmdhandler.NewMessageHandler(rh.list))
	ch.SetHandler("show", cmdhandler.NewMessageHandler(rh.show))
	ch.SetHandler("signup", cmdhandler.NewMessageHandler(rh.signup))
	ch.SetHandler("su", cmdhandler.NewMessageHandler(rh.signup))
	ch.SetHandler("withdraw", cmdhandler.NewMessageHandler(rh.withdraw))
	ch.SetHandler("wd", cmdhandler.NewMessageHandler(rh.withdraw))

	return ch, nil
}

type configDependencies interface {
	Logger() log.Logger
	GuildAPI() storage.GuildAPI
}

// ConfigHandler TODOC
func ConfigHandler(deps configDependencies, versionStr string, opts Options) (*cmdhandler.CommandHandler, error) {
	p := parser.NewParser(parser.Options{
		CmdIndicator: opts.CmdIndicator,
	})

	ch, err := cmdhandler.NewCommandHandler(p, cmdhandler.Options{
		NoHelpOnUnknownCommands: true,
	})
	if err != nil {
		return nil, err
	}

	cch, err := ConfigCommandHandler(deps, versionStr, fmt.Sprintf("%sconfig", opts.CmdIndicator))
	if err != nil {
		return nil, err
	}

	ch.SetHandler("config-su", cch)
	// disable help for config
	ch.SetHandler("help", cmdhandler.NewMessageHandler(func(msg cmdhandler.Message) (cmdhandler.Response, error) {
		r := &cmdhandler.SimpleEmbedResponse{}
		return r, parser.ErrUnknownCommand
	}))

	return ch, nil
}

type adminDependencies interface {
	Logger() log.Logger
	GuildAPI() storage.GuildAPI
	TrialAPI() storage.TrialAPI
	BotSession() *session.Session
}

// AdminHandler TODOC
func AdminHandler(deps adminDependencies, versionStr string, opts Options) (*cmdhandler.CommandHandler, error) {
	p := parser.NewParser(parser.Options{
		CmdIndicator: opts.CmdIndicator,
	})

	ch, err := cmdhandler.NewCommandHandler(p, cmdhandler.Options{
		NoHelpOnUnknownCommands: true,
	})
	if err != nil {
		return nil, err
	}

	ach, err := AdminCommandHandler(deps, fmt.Sprintf("%sadmin", opts.CmdIndicator))
	if err != nil {
		return nil, err
	}

	ch.SetHandler("admin", ach)
	// disable help for admin
	ch.SetHandler("help", cmdhandler.NewMessageHandler(func(msg cmdhandler.Message) (cmdhandler.Response, error) {
		r := &cmdhandler.SimpleEmbedResponse{}
		return r, parser.ErrUnknownCommand
	}))

	return ch, nil
}
