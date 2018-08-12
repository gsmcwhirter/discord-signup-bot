package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gsmcwhirter/discord-bot-lib/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/snowflake"
	"github.com/gsmcwhirter/discord-bot-lib/util"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
	"github.com/gsmcwhirter/go-util/parser"
	"github.com/pkg/errors"
)

type adminCommands struct {
	preCommand string
	deps       adminDependencies
}

func (c *adminCommands) list(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.EmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), false)
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

func (c *adminCommands) create(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	var trialName string
	var settings string

	argParts := strings.SplitN(strings.TrimSpace(msg.Contents()), " ", 2)
	trialName = argParts[0]
	if len(argParts) < 2 {
		settings = ""
	} else {
		settings = argParts[1]
	}

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID().ToString())
	if err != nil {
		return r, err
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), true)
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

	if err = t.SaveTrial(trial); err != nil {
		return r, errors.Wrap(err, "could not save trial")
	}

	if err = t.Commit(); err != nil {
		return r, errors.Wrap(err, "could not save trial")
	}

	r.Description = fmt.Sprintf("Trial %s created successfully", trialName)

	return r, nil
}

func (c *adminCommands) edit(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	var trialName string
	var settings string

	argParts := strings.SplitN(strings.TrimSpace(msg.Contents()), " ", 2)
	trialName = argParts[0]
	if len(argParts) < 2 {
		settings = ""
	} else {
		settings = argParts[1]
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), true)
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

	if err = t.SaveTrial(trial); err != nil {
		return r, errors.Wrap(err, "could not save trial")
	}

	if err = t.Commit(); err != nil {
		return r, errors.Wrap(err, "could not save trial")
	}

	r.Description = fmt.Sprintf("Trial %s edited successfully", trialName)

	return r, nil
}

func (c *adminCommands) open(msg cmdhandler.Message) (cmdhandler.Response, error) {
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

	trial.SetState(storage.TrialStateOpen)

	if err = t.SaveTrial(trial); err != nil {
		return r, errors.Wrap(err, "could not open trial")
	}

	if err = t.Commit(); err != nil {
		return r, errors.Wrap(err, "could not open trial")
	}

	r.Description = fmt.Sprintf("Opened trial %s", trialName)

	return r, nil
}

func (c *adminCommands) close(msg cmdhandler.Message) (cmdhandler.Response, error) {
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

	trial.SetState(storage.TrialStateClosed)

	if err = t.SaveTrial(trial); err != nil {
		return r, errors.Wrap(err, "could not close trial")
	}

	if err = t.Commit(); err != nil {
		return r, errors.Wrap(err, "could not close trial")
	}

	r.Description = fmt.Sprintf("Closed trial %s", trialName)

	return r, nil
}

func (c *adminCommands) delete(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer util.CheckDefer(t.Rollback)

	trialName := strings.TrimSpace(msg.Contents())

	if err = t.DeleteTrial(trialName); err != nil {
		return r, errors.Wrap(err, "could not delete trial")
	}

	if err = t.Commit(); err != nil {
		return r, errors.Wrap(err, "could not delete trial")
	}

	r.Description = fmt.Sprintf("Deleted trial %s", trialName)

	return r, nil
}

func (c *adminCommands) announce(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	parts := strings.SplitN(strings.TrimSpace(msg.Contents()), " ", 2)
	trialName := parts[0]
	phrase := ""
	if len(parts) > 1 {
		phrase = parts[1]
	}

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID().ToString())
	if err != nil {
		return r, err
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), false)
	if err != nil {
		return r, err
	}
	defer util.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	sessionGuild, err := c.deps.BotSession().Guild(msg.GuildID())
	if err != nil {
		return r, err
	}

	var signupCid snowflake.Snowflake
	var announceCid snowflake.Snowflake

	if scID, ok := sessionGuild.ChannelWithName(trial.GetSignupChannel()); ok {
		signupCid = scID
	}

	if acID, ok := sessionGuild.ChannelWithName(trial.GetAnnounceChannel()); ok {
		announceCid = acID
	}

	roles := trial.GetRoleCounts()
	roleStrs := make([]string, 0, len(roles))
	for _, rc := range roles {
		roleStrs = append(roleStrs, fmt.Sprintf("%s: %d", rc.GetRole(), rc.GetCount()))
	}

	toStr := "@everyone"
	if gsettings.AnnounceTo != "" {
		toStr = gsettings.AnnounceTo
	}

	r2 := &cmdhandler.EmbedResponse{
		To:          fmt.Sprintf("%s %s", toStr, phrase),
		ToChannel:   announceCid,
		Title:       fmt.Sprintf("Signups are open for %s", trial.GetName()),
		Description: trial.GetDescription(),
		Fields: []cmdhandler.EmbedField{
			{
				Name: "Roles Requested",
				Val:  fmt.Sprintf("```\n%s\n```\n", strings.Join(roleStrs, "\n")),
			},
		},
	}

	if signupCid != 0 {
		r2.Fields = append(r2.Fields, cmdhandler.EmbedField{
			Name: "Signup Channel",
			Val:  cmdhandler.ChannelMentionString(signupCid),
		})
	}

	return r2, nil
}

func (c *adminCommands) grouping(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	parts := strings.SplitN(strings.TrimSpace(msg.Contents()), " ", 2)
	trialName := parts[0]
	phrase := "Grouping now!"
	if len(parts) > 1 {
		phrase = parts[1]
	}

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID().ToString())
	if err != nil {
		return r, err
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), false)
	if err != nil {
		return r, err
	}
	defer util.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	sessionGuild, err := c.deps.BotSession().Guild(msg.GuildID())
	if err != nil {
		return r, err
	}

	var announceCid snowflake.Snowflake
	if acID, ok := sessionGuild.ChannelWithName(trial.GetAnnounceChannel()); ok {
		announceCid = acID
	}

	toStr := "@everyone"
	if gsettings.AnnounceTo != "" {
		toStr = gsettings.AnnounceTo
	}

	r2 := formatTrialDisplay(trial, false)
	r2.To = fmt.Sprintf("%s %s", toStr, phrase)
	r2.ToChannel = announceCid

	return r2, nil
}

func (c *adminCommands) signup(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	parts := strings.SplitN(strings.TrimSpace(msg.Contents()), " ", 3)
	if len(parts) != 3 {
		return r, errors.New("not enough arguments (need `user-mention trial-name role`")
	}
	userMention := parts[0]
	trialName := parts[1]
	role := parts[2]

	if !cmdhandler.IsUserMention(userMention) {
		return r, errors.New("you must mention the user you are trying to sign up (@...)")
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

	if trial.GetState() != storage.TrialStateOpen {
		return r, errors.New("cannot sign up for a closed trial")
	}

	sessionGuild, err := c.deps.BotSession().Guild(msg.GuildID())
	if err != nil {
		return r, err
	}

	var signupCid snowflake.Snowflake
	if scID, ok := sessionGuild.ChannelWithName(trial.GetSignupChannel()); ok {
		signupCid = scID
	}

	overflow, err := signupUser(trial, userMention, role)
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
		descStr = fmt.Sprintf("Signed up as OVERFLOW for %s in %s by %s", role, trialName, cmdhandler.UserMentionString(msg.UserID()))
	} else {
		descStr = fmt.Sprintf("Signed up for %s in %s by %s", role, trialName, cmdhandler.UserMentionString(msg.UserID()))
	}

	r.To = userMention
	r.ToChannel = signupCid
	r.Description = descStr

	return r, nil
}

func (c *adminCommands) withdraw(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	parts := strings.SplitN(strings.TrimSpace(msg.Contents()), " ", 2)
	if len(parts) != 2 {
		return r, errors.New("not enough arguments (need `user-mention trial-name`")
	}
	userMention := parts[0]
	trialName := parts[1]

	if !cmdhandler.IsUserMention(userMention) {
		return r, errors.New("you must mention the user you are trying to withdraw (@...)")
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

	if trial.GetState() != storage.TrialStateOpen {
		return r, errors.New("cannot withdraw from a closed trial")
	}

	sessionGuild, err := c.deps.BotSession().Guild(msg.GuildID())
	if err != nil {
		return r, err
	}

	var signupCid snowflake.Snowflake
	if scID, ok := sessionGuild.ChannelWithName(trial.GetSignupChannel()); ok {
		signupCid = scID
	}

	trial.RemoveSignup(userMention)

	if err = t.SaveTrial(trial); err != nil {
		return r, errors.Wrap(err, "could not save trial withdraw")
	}

	if err = t.Commit(); err != nil {
		return r, errors.Wrap(err, "could not save trial withdraw")
	}

	r.To = userMention
	r.ToChannel = signupCid
	r.Description = fmt.Sprintf("Withdrawn from %s by %s", trialName, cmdhandler.UserMentionString(msg.UserID()))

	return r, nil
}

// AdminCommandHandler TODOC
func AdminCommandHandler(deps adminDependencies, preCommand string) (*cmdhandler.CommandHandler, error) {
	p := parser.NewParser(parser.Options{
		CmdIndicator: " ",
	})
	cc := adminCommands{
		preCommand: preCommand,
		deps:       deps,
	}

	ch, err := cmdhandler.NewCommandHandler(p, cmdhandler.Options{
		PreCommand:          preCommand,
		Placeholder:         "action",
		HelpOnEmptyCommands: true,
	})
	if err != nil {
		return nil, err
	}

	ch.SetHandler("list", cmdhandler.NewMessageHandler(cc.list))
	ch.SetHandler("create", cmdhandler.NewMessageHandler(cc.create))
	ch.SetHandler("edit", cmdhandler.NewMessageHandler(cc.edit))
	ch.SetHandler("open", cmdhandler.NewMessageHandler(cc.open))
	ch.SetHandler("close", cmdhandler.NewMessageHandler(cc.close))
	ch.SetHandler("delete", cmdhandler.NewMessageHandler(cc.delete))
	ch.SetHandler("announce", cmdhandler.NewMessageHandler(cc.announce))
	ch.SetHandler("grouping", cmdhandler.NewMessageHandler(cc.grouping))
	ch.SetHandler("signup", cmdhandler.NewMessageHandler(cc.signup))
	ch.SetHandler("su", cmdhandler.NewMessageHandler(cc.signup))
	ch.SetHandler("withdraw", cmdhandler.NewMessageHandler(cc.withdraw))
	ch.SetHandler("wd", cmdhandler.NewMessageHandler(cc.withdraw))

	return ch, nil
}
