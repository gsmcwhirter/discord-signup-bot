package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gsmcwhirter/discord-bot-lib/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/util"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
	"github.com/gsmcwhirter/go-util/parser"
	"github.com/pkg/errors"
)

type adminCommands struct {
	preCommand string
	deps       adminDependencies
}

func (c *adminCommands) list(user, guild, args string) (cmdhandler.Response, error) {
	r := &cmdhandler.EmbedResponse{
		To: user,
	}

	t, err := c.deps.TrialAPI().NewTransaction(guild, false)
	if err != nil {
		return r, err
	}
	defer util.CheckDefer(t.Rollback)

	trials := t.GetTrials()
	tNamesOpen := make([]string, 0, len(trials))
	tNamesClosed := make([]string, 0, len(trials))
	for _, trial := range trials {
		if trial.GetState() == storage.TrialStateClosed {
			tNamesClosed = append(tNamesClosed, fmt.Sprintf("%s (#%s)", trial.GetName(), trial.GetSignupChannel()))
		} else {
			tNamesOpen = append(tNamesOpen, fmt.Sprintf("%s (#%s)", trial.GetName(), trial.GetSignupChannel()))
		}
	}
	sort.Strings(tNamesOpen)
	sort.Strings(tNamesClosed)

	r.Fields = []cmdhandler.EmbedField{
		{
			Name: "*Available Trials*",
			Val:  fmt.Sprintf("```\n%s\n```\n", strings.Join(tNamesOpen, "\n")),
		},
		{
			Name: "*Closed Trials*",
			Val:  fmt.Sprintf("```\n%s\n```\n", strings.Join(tNamesClosed, "\n")),
		},
	}

	return r, nil
}

func (c *adminCommands) create(user, guild, args string) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: user,
	}

	var trialName string
	var settings string

	argParts := strings.SplitN(strings.TrimSpace(args), " ", 2)
	trialName = argParts[0]
	if len(argParts) < 2 {
		settings = ""
	} else {
		settings = argParts[1]
	}

	gsettings, err := getGuildSettings(c.deps.GuildAPI(), guild)
	if err != nil {
		return r, err
	}

	t, err := c.deps.TrialAPI().NewTransaction(guild, true)
	if err != nil {
		return r, err
	}
	defer util.CheckDefer(t.Rollback)

	trial, err := t.AddTrial(trialName)
	if err != nil {
		return r, err
	}

	settingMap, err := parseSettingDescriptionArgs(settings)
	if err != nil {
		return r, err
	}

	trial.SetName(trialName)
	trial.SetDescription(settingMap["description"])
	trial.SetState(storage.TrialStateOpen)

	if v, ok := settingMap["announcechannel"]; !ok {
		trial.SetAnnounceChannel(gsettings.AnnounceChannel)
	} else {
		trial.SetAnnounceChannel(v)
	}

	if v, ok := settingMap["signupchannel"]; !ok {
		trial.SetSignupChannel(gsettings.SignupChannel)
	} else {
		trial.SetSignupChannel(v)
	}

	roleCtMap, err := parseRolesString(settingMap["roles"])
	if err != nil {
		return r, err
	}
	for role, ct := range roleCtMap {
		if ct != 0 {
			trial.SetRoleCount(role, ct)
		}
	}

	err = t.SaveTrial(trial)
	if err != nil {
		return r, errors.Wrap(err, "could not save trial")
	}

	err = t.Commit()
	if err != nil {
		return r, errors.Wrap(err, "could not save trial")
	}

	r.Description = fmt.Sprintf("Trial %s created successfully", trialName)

	return r, nil
}

func (c *adminCommands) edit(user, guild, args string) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: user,
	}

	var trialName string
	var settings string

	argParts := strings.SplitN(strings.TrimSpace(args), " ", 2)
	trialName = argParts[0]
	if len(argParts) < 2 {
		settings = ""
	} else {
		settings = argParts[1]
	}

	t, err := c.deps.TrialAPI().NewTransaction(guild, true)
	if err != nil {
		return r, err
	}
	defer util.CheckDefer(t.Rollback)

	trial, err := t.AddTrial(trialName)
	if err != nil {
		return r, err
	}

	settingMap, err := parseSettingDescriptionArgs(settings)
	if err != nil {
		return r, err
	}

	if v, ok := settingMap["description"]; ok {
		trial.SetDescription(v)
	}

	if v, ok := settingMap["announcechannel"]; ok {
		trial.SetAnnounceChannel(v)
	}

	if v, ok := settingMap["signupchannel"]; ok {
		trial.SetSignupChannel(v)
	}

	roleCtMap, err := parseRolesString(settingMap["roles"])
	if err != nil {
		return r, err
	}
	for role, ct := range roleCtMap {
		if ct == 0 {
			trial.RemoveRole(role)
		} else {
			trial.SetRoleCount(role, ct)
		}
	}

	err = t.SaveTrial(trial)
	if err != nil {
		return r, errors.Wrap(err, "could not save trial")
	}

	err = t.Commit()
	if err != nil {
		return r, errors.Wrap(err, "could not save trial")
	}

	r.Description = fmt.Sprintf("Trial %s created successfully", trialName)

	return r, nil
}

func (c *adminCommands) open(user, guild, args string) (cmdhandler.Response, error) {
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

	trial.SetState(storage.TrialStateOpen)

	err = t.SaveTrial(trial)
	if err != nil {
		return r, errors.Wrap(err, "could not open trial")
	}

	err = t.Commit()
	if err != nil {
		return r, errors.Wrap(err, "could not open trial")
	}

	r.Description = fmt.Sprintf("Opened trial %s", trialName)

	return r, nil
}

func (c *adminCommands) close(user, guild, args string) (cmdhandler.Response, error) {
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

	trial.SetState(storage.TrialStateClosed)

	err = t.SaveTrial(trial)
	if err != nil {
		return r, errors.Wrap(err, "could not close trial")
	}

	err = t.Commit()
	if err != nil {
		return r, errors.Wrap(err, "could not close trial")
	}

	r.Description = fmt.Sprintf("Closed trial %s", trialName)

	return r, nil
}

// func (c *adminCommands) delete(user, guild, args string) (cmdhandler.Response, error) {
// 	r := &cmdhandler.SimpleEmbedResponse{
// 		To: user,
// 	}

// 	return r, nil
// }

func (c *adminCommands) announce(user, guild, args string) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: user,
	}

	return r, nil
}

// AdminCommandHandler TODOC
func AdminCommandHandler(deps adminDependencies, preCommand string) *cmdhandler.CommandHandler {
	p := parser.NewParser(parser.Options{
		CmdIndicator: " ",
	})
	cc := adminCommands{
		preCommand: preCommand,
		deps:       deps,
	}

	ch := cmdhandler.NewCommandHandler(p, cmdhandler.Options{
		PreCommand:          preCommand,
		Placeholder:         "action",
		HelpOnEmptyCommands: true,
	})
	ch.SetHandler("list", cmdhandler.NewLineHandler(cc.list))
	ch.SetHandler("create", cmdhandler.NewLineHandler(cc.create))
	ch.SetHandler("edit", cmdhandler.NewLineHandler(cc.edit))
	ch.SetHandler("open", cmdhandler.NewLineHandler(cc.open))
	ch.SetHandler("close", cmdhandler.NewLineHandler(cc.close))
	// ch.SetHandler("delete", cmdhandler.NewLineHandler(cc.delete))
	ch.SetHandler("announce", cmdhandler.NewLineHandler(cc.announce))

	return ch
}
