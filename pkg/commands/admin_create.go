package commands

import (
	"fmt"

	"github.com/gsmcwhirter/go-util/v3/deferutil"
	"github.com/gsmcwhirter/go-util/v3/errors"
	"github.com/gsmcwhirter/go-util/v3/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v8/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v8/logging"
)

func (c *adminCommands) create(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling adminCommand", "command", "create", "args", msg.Contents())

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		level.Info(logger).Message("command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return nil, msghandler.ErrUnauthorized
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

	level.Info(logger).Message("trial created", "trial_name", trialName)
	r.Description = fmt.Sprintf("Event %q created successfully", trialName)

	return r, nil
}
