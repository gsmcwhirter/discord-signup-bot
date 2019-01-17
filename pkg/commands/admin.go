package commands

import (
	"fmt"
	"sort"
	"strings"

	multierror "github.com/hashicorp/go-multierror"

	"github.com/go-kit/kit/log/level"
	"github.com/gsmcwhirter/discord-bot-lib/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/logging"
	"github.com/gsmcwhirter/discord-bot-lib/snowflake"
	"github.com/gsmcwhirter/go-util/deferutil"
	"github.com/gsmcwhirter/go-util/parser"
	"github.com/pkg/errors"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

// ErrGuildNotFound is the error returned when a guild is not known about
// in a BotSession
var ErrGuildNotFound = errors.New("guild not found")

type adminCommands struct {
	preCommand string
	deps       adminDependencies
}

func (c *adminCommands) list(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.EmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling adminCommand", "command", "list")

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		_ = level.Info(logger).Log("message", "command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, msghandler.ErrNoResponse
	}

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

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

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling adminCommand", "command", "create", "args", msg.Contents())

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		_ = level.Info(logger).Log("message", "command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, msghandler.ErrNoResponse
	}

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 1 {
		return r, errors.New("need event name")
	}

	trialName := msg.Contents()[0]
	settings := msg.Contents()[1:]

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

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

	if v, ok := settingMap["announceto"]; ok {
		trial.SetAnnounceTo(v)
	}

	if v, ok := settingMap["signupchannel"]; !ok {
		trial.SetSignupChannel(gsettings.SignupChannel)
	} else {
		trial.SetSignupChannel(v)
	}

	roleCtEmoList, err := parseRolesString(settingMap["roles"])
	if err != nil {
		return r, err
	}
	for _, rce := range roleCtEmoList {
		if rce.ct != 0 {
			trial.SetRoleCount(rce.role, rce.emo, rce.ct)
		}
	}

	if err = t.SaveTrial(trial); err != nil {
		return r, errors.Wrap(err, "could not save event")
	}

	if err = t.Commit(); err != nil {
		return r, errors.Wrap(err, "could not save event")
	}

	_ = level.Info(logger).Log("message", "trial created", "trial_name", trialName)
	r.Description = fmt.Sprintf("Event %q created successfully", trialName)

	return r, nil
}

func (c *adminCommands) edit(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling adminCommand", "command", "edit", "args", msg.Contents())

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		_ = level.Info(logger).Log("message", "command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, msghandler.ErrNoResponse
	}

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 1 {
		return r, errors.New("need event name")
	}

	trialName := msg.Contents()[0]
	settings := msg.Contents()[1:]

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

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

	if v, ok := settingMap["announceto"]; ok {
		trial.SetAnnounceTo(v)
	}

	if v, ok := settingMap["signupchannel"]; ok {
		trial.SetSignupChannel(v)
	}

	roleCtEmoList, err := parseRolesString(settingMap["roles"])
	if err != nil {
		return r, err
	}
	for _, rce := range roleCtEmoList {
		if rce.ct == 0 {
			trial.RemoveRole(rce.role)
		} else {
			trial.SetRoleCount(rce.role, rce.emo, rce.ct)
		}
	}

	if err = t.SaveTrial(trial); err != nil {
		return r, errors.Wrap(err, "could not save event")
	}

	if err = t.Commit(); err != nil {
		return r, errors.Wrap(err, "could not save event")
	}

	_ = level.Info(logger).Log("message", "trial edited", "trial_name", trialName)
	r.Description = fmt.Sprintf("Trial %s edited successfully", trialName)

	return r, nil
}

func (c *adminCommands) open(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling adminCommand", "command", "open", "args", msg.Contents())

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		_ = level.Info(logger).Log("message", "command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, msghandler.ErrNoResponse
	}

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 1 {
		return r, errors.New("need event name")
	}

	if len(msg.Contents()) > 1 {
		return r, errors.New("too many arguments")
	}

	trialName := msg.Contents()[0]

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	trial.SetState(storage.TrialStateOpen)

	if err = t.SaveTrial(trial); err != nil {
		return r, errors.Wrap(err, "could not open event")
	}

	if err = t.Commit(); err != nil {
		return r, errors.Wrap(err, "could not open event")
	}

	_ = level.Info(logger).Log("message", "trial opened", "trial_name", trialName)
	r.Description = fmt.Sprintf("Opened event %q", trialName)

	return r, nil
}

func (c *adminCommands) close(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling adminCommand", "command", "close", "args", msg.Contents())

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		_ = level.Info(logger).Log("message", "command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, msghandler.ErrNoResponse
	}

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 1 {
		return r, errors.New("need event name")
	}

	if len(msg.Contents()) > 1 {
		return r, errors.New("too many arguments")
	}

	trialName := msg.Contents()[0]

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	trial.SetState(storage.TrialStateClosed)

	if err = t.SaveTrial(trial); err != nil {
		return r, errors.Wrap(err, "could not close event")
	}

	if err = t.Commit(); err != nil {
		return r, errors.Wrap(err, "could not close event")
	}

	_ = level.Info(logger).Log("message", "trial closed", "trial_name", trialName)
	r.Description = fmt.Sprintf("Closed event %q", trialName)

	return r, nil
}

func (c *adminCommands) delete(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling adminCommand", "command", "delete", "args", msg.Contents())

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		_ = level.Info(logger).Log("message", "command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, msghandler.ErrNoResponse
	}

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 1 {
		return r, errors.New("need event name")
	}

	if len(msg.Contents()) > 1 {
		return r, errors.New("too many arguments")
	}

	trialName := msg.Contents()[0]

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	if err = t.DeleteTrial(trialName); err != nil {
		return r, errors.Wrap(err, "could not delete event")
	}

	if err = t.Commit(); err != nil {
		return r, errors.Wrap(err, "could not delete event")
	}

	_ = level.Info(logger).Log("message", "trial deleted", "trial_name", trialName)
	r.Description = fmt.Sprintf("Deleted event %q", trialName)

	return r, nil
}

func (c *adminCommands) announce(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling adminCommand", "command", "announce", "args", msg.Contents())

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		_ = level.Info(logger).Log("message", "command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, msghandler.ErrNoResponse
	}

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 1 {
		return r, errors.New("need event name")
	}

	trialName := msg.Contents()[0]
	phrase := strings.Join(msg.Contents()[1:], " ")

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	sessionGuild, ok := c.deps.BotSession().Guild(msg.GuildID())
	if !ok {
		return r, ErrGuildNotFound
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

	var toStr string
	switch {
	case trial.GetAnnounceTo() != "":
		toStr = trial.GetAnnounceTo()
	case gsettings.AnnounceTo != "":
		toStr = gsettings.AnnounceTo
	default:
		toStr = "@everyone"
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

	_ = level.Info(logger).Log("message", "trial announced", "trial_name", trialName, "announce_channel", r2.ToChannel.ToString(), "announce_to", r2.To)

	return r2, nil
}

func (c *adminCommands) show(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling adminCommand", "command", "show", "args", msg.Contents())

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		_ = level.Info(logger).Log("message", "command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, msghandler.ErrNoResponse
	}

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 1 {
		return r, errors.New("need event name")
	}

	if len(msg.Contents()) > 1 {
		return r, errors.New("too many arguments")
	}

	trialName := msg.Contents()[0]

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	r.Description = trial.PrettySettings()

	_ = level.Info(logger).Log("message", "trial shown", "trial_name", trialName)

	return r, nil
}

func (c *adminCommands) grouping(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling adminCommand", "command", "grouping", "args", msg.Contents())

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		_ = level.Info(logger).Log("message", "command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, msghandler.ErrNoResponse
	}

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 1 {
		return r, errors.New("need event name")
	}

	trialName := msg.Contents()[0]
	phrase := fmt.Sprintf("Grouping now for %s!", trialName)
	if len(msg.Contents()) > 1 {
		phrase = strings.Join(msg.Contents()[1:], " ")
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	sessionGuild, ok := c.deps.BotSession().Guild(msg.GuildID())
	if !ok {
		return r, ErrGuildNotFound
	}

	var announceCid snowflake.Snowflake
	if acID, ok := sessionGuild.ChannelWithName(trial.GetAnnounceChannel()); ok {
		announceCid = acID
	}

	roleCounts := trial.GetRoleCounts() // already sorted by name
	signups := trial.GetSignups()

	userMentions := make([]string, 0, len(signups))

	for _, rc := range roleCounts {
		suNames, ofNames := getTrialRoleSignups(signups, rc)

		for _, u := range suNames {
			userMentions = append(userMentions, u)
		}

		for _, u := range ofNames {
			userMentions = append(userMentions, u)
		}
	}

	fmt.Printf("** %v\n", userMentions)

	toStr := strings.Join(userMentions, ", ")

	r.To = fmt.Sprintf("%s\n\n%s", toStr, phrase)
	r.ToChannel = announceCid

	_ = level.Info(logger).Log("message", "trial grouping", "trial_name", trialName, "announce_channel", r.ToChannel.ToString(), "announce_to", r.To)

	return r, nil
}

func (c *adminCommands) signup(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling adminCommand", "command", "signup", "args", msg.Contents())

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		_ = level.Info(logger).Log("message", "command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, msghandler.ErrNoResponse
	}

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 3 {
		return r, errors.New("not enough arguments (need `event-name role user-mention(s)`")
	}

	trialName := msg.Contents()[0]
	role := msg.Contents()[1]
	userMentions := make([]string, 0, len(msg.Contents())-2)

	for _, m := range msg.Contents()[2:] {
		if !cmdhandler.IsUserMention(m) {
			_ = level.Warn(logger).Log("message", "skipping signup user", "reason", "not user mention")
			continue
		}

		m, err = cmdhandler.ForceUserNicknameMention(m)
		if err != nil {
			_ = level.Warn(logger).Log("message", "skipping signup user", "reason", err)
			continue
		}

		userMentions = append(userMentions, m)
	}

	if len(userMentions) == 0 {
		return r, errors.New("you must mention one or more users that you are trying to sign up (@...)")
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	if trial.GetState() != storage.TrialStateOpen {
		return r, errors.New("cannot sign up for a closed event")
	}

	sessionGuild, ok := c.deps.BotSession().Guild(msg.GuildID())
	if !ok {
		return r, ErrGuildNotFound
	}

	var signupCid snowflake.Snowflake
	if scID, ok := sessionGuild.ChannelWithName(trial.GetSignupChannel()); ok {
		signupCid = scID
	}

	overflows := make([]bool, len(userMentions))
	regularUsers := make([]string, 0, len(userMentions))
	overflowUsers := make([]string, 0, len(userMentions))

	for i, userMention := range userMentions {
		var serr error
		overflows[i], serr = signupUser(trial, userMention, role)
		if serr != nil {
			err = multierror.Append(err, serr)
			continue
		}

		if overflows[i] {
			overflowUsers = append(overflowUsers, userMention)
		} else {
			regularUsers = append(regularUsers, userMention)
		}
	}

	if err != nil {
		return r, err
	}

	if err = t.SaveTrial(trial); err != nil {
		return r, errors.Wrap(err, "could not save event signup")
	}

	if err = t.Commit(); err != nil {
		return r, errors.Wrap(err, "could not save event signup")
	}

	descStr := fmt.Sprintf("Signed up for %s in %s by %s\n\n", role, trialName, cmdhandler.UserMentionString(msg.UserID()))
	if len(regularUsers) > 0 {
		descStr += fmt.Sprintf("**Main Group:** %s\n", strings.Join(regularUsers, ", "))
	}
	if len(overflowUsers) > 0 {
		descStr += fmt.Sprintf("**Overflow:** %s\n", strings.Join(overflowUsers, ", "))
	}

	if gsettings.ShowAfterSignup == "true" {
		_ = level.Debug(logger).Log("message", "auto-show after signup", "trial_name", trialName)

		r2 := formatTrialDisplay(trial, true)
		r2.To = strings.Join(userMentions, ", ")
		r2.ToChannel = signupCid
		r2.Description = fmt.Sprintf("%s\n\n%s", descStr, r2.Description)
		return r2, nil
	}

	r.To = strings.Join(userMentions, ", ")
	r.ToChannel = signupCid
	r.Description = descStr

	_ = level.Info(logger).Log("message", "admin signup complete", "trial_name", trialName, "signup_users", userMentions, "overflows", overflows, "role", role, "signup_channel", r.ToChannel.ToString())

	return r, nil
}

func (c *adminCommands) withdraw(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling adminCommand", "command", "withdraw", "args", msg.Contents())

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		_ = level.Info(logger).Log("message", "command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, msghandler.ErrNoResponse
	}

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 2 {
		return r, errors.New("not enough arguments (need `trial-name user-mention(s)`")
	}
	trialName := msg.Contents()[0]
	userMentions := make([]string, 0, len(msg.Contents())-2)

	for _, m := range msg.Contents()[1:] {
		if !cmdhandler.IsUserMention(m) {
			_ = level.Warn(logger).Log("message", "skipping withdraw user", "reason", "not user mention")
			continue
		}

		m, err = cmdhandler.ForceUserNicknameMention(m)
		if err != nil {
			_ = level.Warn(logger).Log("message", "skipping withdraw user", "reason", err)
			continue
		}

		userMentions = append(userMentions, m)
	}

	if len(userMentions) == 0 {
		return r, errors.New("you must mention one or more users that you are trying to withdraw (@...)")
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	if trial.GetState() != storage.TrialStateOpen {
		return r, errors.New("cannot withdraw from a closed event")
	}

	sessionGuild, ok := c.deps.BotSession().Guild(msg.GuildID())
	if !ok {
		return r, ErrGuildNotFound
	}

	var signupCid snowflake.Snowflake
	if scID, ok := sessionGuild.ChannelWithName(trial.GetSignupChannel()); ok {
		signupCid = scID
	}

	for _, m := range userMentions {
		userAcctMention, werr := cmdhandler.ForceUserAccountMention(m)
		if err != nil {
			err = multierror.Append(err, werr)
			continue
		}

		trial.RemoveSignup(userAcctMention)
		trial.RemoveSignup(m)
	}

	if err != nil {
		return r, err
	}

	if err = t.SaveTrial(trial); err != nil {
		return r, errors.Wrap(err, "could not save event withdraw")
	}

	if err = t.Commit(); err != nil {
		return r, errors.Wrap(err, "could not save event withdraw")
	}

	descStr := fmt.Sprintf("Withdrawn from %s by %s", trialName, cmdhandler.UserMentionString(msg.UserID()))

	if gsettings.ShowAfterWithdraw == "true" {
		_ = level.Debug(logger).Log("message", "auto-show after withdraw", "trial_name", trialName)

		r2 := formatTrialDisplay(trial, true)
		r2.To = strings.Join(userMentions, ", ")
		r2.ToChannel = signupCid
		r2.Description = fmt.Sprintf("%s\n\n%s", descStr, r2.Description)
		return r2, nil
	}

	r.To = strings.Join(userMentions, ", ")
	r.ToChannel = signupCid
	r.Description = descStr

	_ = level.Info(logger).Log("message", "admin withdraw complete", "trial_name", trialName, "withdraw_users", userMentions, "signup_channel", r.ToChannel.ToString())

	return r, nil
}

func (c *adminCommands) clear(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling adminCommand", "command", "clear", "args", msg.Contents())

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		_ = level.Info(logger).Log("message", "command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, msghandler.ErrNoResponse
	}

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 1 {
		return r, errors.New("need event name")
	}

	if len(msg.Contents()) > 1 {
		return r, errors.New("too many arguments")
	}

	trialName := msg.Contents()[0]

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	trial.ClearSignups()

	if err = t.SaveTrial(trial); err != nil {
		return r, errors.Wrap(err, "could not save event")
	}

	if err = t.Commit(); err != nil {
		return r, errors.Wrap(err, "could not save event")
	}

	_ = level.Info(logger).Log("message", "trial cleared", "trial_name", trialName)
	r.Description = fmt.Sprintf("Event %q cleared successfully", trialName)

	return r, nil
}

// AdminCommandHandler creates a new command handler for !admin commands
func AdminCommandHandler(deps adminDependencies, preCommand string) (*cmdhandler.CommandHandler, error) {
	p := parser.NewParser(parser.Options{
		CmdIndicator: "",
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
	ch.SetHandler("clear", cmdhandler.NewMessageHandler(cc.clear))
	ch.SetHandler("show", cmdhandler.NewMessageHandler(cc.show))

	return ch, nil
}
