package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gsmcwhirter/discord-bot-lib/cmdhandler"
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

func (c *rootCommands) version(user, guild, args string) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To:          user,
		Description: c.versionStr,
	}
	return r, nil
}

func (c *rootCommands) list(user, guild, args string) (cmdhandler.Response, error) {
	r := &cmdhandler.EmbedResponse{
		To: user,
	}

	t, err := c.deps.TrialAPI().NewTransaction(guild, false)
	if err != nil {
		return r, err
	}
	defer util.CheckDefer(t.Rollback)

	trials := t.GetTrials()
	tNames := make([]string, 0, len(trials))
	for _, trial := range trials {
		if trial.GetState() != storage.TrialStateClosed {
			tNames = append(tNames, fmt.Sprintf("%s (#%s)", trial.GetName(), trial.GetSignupChannel()))
		}
	}
	sort.Strings(tNames)

	r.Fields = []cmdhandler.EmbedField{
		{
			Name: "*Available Trials*",
			Val:  fmt.Sprintf("```\n%s\n```\n", strings.Join(tNames, "\n")),
		},
	}

	return r, nil
}

func (c *rootCommands) show(user, guild, args string) (cmdhandler.Response, error) {
	r := &cmdhandler.EmbedResponse{
		To: user,
	}

	trialName := strings.TrimSpace(args)

	t, err := c.deps.TrialAPI().NewTransaction(guild, false)
	if err != nil {
		return r, err
	}
	defer util.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	r.Title = fmt.Sprintf("__%s__ (%s)", trial.GetName(), string(trial.GetState()))
	r.Description = trial.GetDescription()
	r.Fields = []cmdhandler.EmbedField{}

	roleCounts := trial.GetRoleCounts() // already sorted by name
	signups := trial.GetSignups()

	for _, rc := range roleCounts {
		lowerRole := strings.ToLower(rc.GetRole())
		suNames := make([]string, 0, len(signups))
		ofNames := make([]string, 0, len(signups))
		for _, su := range signups {
			if strings.ToLower(su.GetRole()) != lowerRole {
				continue
			}

			if uint64(len(suNames)) < rc.GetCount() {
				suNames = append(suNames, su.GetName())
			} else {
				ofNames = append(ofNames, su.GetName())
			}
		}

		if len(suNames) > 0 {
			r.Fields = append(r.Fields, cmdhandler.EmbedField{
				Name: fmt.Sprintf("*%s* (%d/%d)", rc.GetRole(), len(suNames), rc.GetCount()),
				Val:  strings.Join(suNames, "\n") + "\n",
			})
		} else {
			r.Fields = append(r.Fields, cmdhandler.EmbedField{
				Name: fmt.Sprintf("*%s* (%d/%d)", rc.GetRole(), len(suNames), rc.GetCount()),
				Val:  "(empty)",
			})
		}

		if len(ofNames) > 0 {
			r.Fields = append(r.Fields, cmdhandler.EmbedField{
				Name: fmt.Sprintf("*%s Overflow* (%d)", rc.GetRole(), len(ofNames)),
				Val:  strings.Join(ofNames, "\n") + "\n",
			})
		}
	}

	return r, nil
}

func (c *rootCommands) signup(user, guild, args string) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: user,
	}

	argParts := strings.SplitN(strings.TrimSpace(args), " ", 2)
	if len(argParts) < 2 {
		return r, errors.New("missing role")
	}

	trialName, role := argParts[0], argParts[1]

	t, err := c.deps.TrialAPI().NewTransaction(guild, true)
	if err != nil {
		return r, err
	}
	defer util.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	roleCounts := trial.GetRoleCounts() // already sorted by name
	rc, known := roleCountByName(role, roleCounts)
	if !known {
		return r, errors.New("unknown role")
	}

	signups := trial.GetSignups()
	roleSignups := signupsForRole(role, signups, false)

	trial.AddSignup(user, role)

	err = t.SaveTrial(trial)
	if err != nil {
		return r, errors.Wrap(err, "could not save trial signup")
	}

	err = t.Commit()
	if err != nil {
		return r, errors.Wrap(err, "could not save trial signup")
	}

	if uint64(len(roleSignups)) >= rc.GetCount() {
		r.Description = fmt.Sprintf("Signed up as OVERFLOW for %s in %s", role, trialName)
	} else {
		r.Description = fmt.Sprintf("Signed up for %s in %s", role, trialName)
	}

	return r, nil
}

func (c *rootCommands) withdraw(user, guild, args string) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: user,
	}

	trialName := strings.TrimSpace(args)

	t, err := c.deps.TrialAPI().NewTransaction(guild, true)
	if err != nil {
		return r, err
	}
	defer util.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	trial.RemoveSignup(user)

	err = t.SaveTrial(trial)
	if err != nil {
		return r, errors.Wrap(err, "could not save trial withdraw")
	}

	err = t.Commit()
	if err != nil {
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
	ch.SetHandler("version", cmdhandler.NewLineHandler(rh.version))
	ch.SetHandler("list", cmdhandler.NewLineHandler(rh.list))
	ch.SetHandler("show", cmdhandler.NewLineHandler(rh.show))
	ch.SetHandler("signup", cmdhandler.NewLineHandler(rh.signup))
	ch.SetHandler("withdraw", cmdhandler.NewLineHandler(rh.withdraw))

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

	return ch
}

type adminDependencies interface {
	GuildAPI() storage.GuildAPI
	TrialAPI() storage.TrialAPI
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

	return ch
}
