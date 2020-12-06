package commands

import (
	"fmt"

	"github.com/gsmcwhirter/go-util/v7/deferutil"
	"github.com/gsmcwhirter/go-util/v7/errors"
	"github.com/gsmcwhirter/go-util/v7/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v16/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v16/logging"
)

func (c *adminCommands) create(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "adminCommands.create", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling adminCommand", "command", "create", "args", msg.Contents())

	gsettings, err := storage.GetSettings(msg.Context(), c.deps.GuildAPI(), msg.GuildID())
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

	t, err := c.deps.TrialAPI().NewTransaction(msg.Context(), msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(msg.Context()) })

	trial, err := t.AddTrial(msg.Context(), trialName)
	if err != nil {
		return r, err
	}

	settingMap, err := parseSettingDescriptionArgs(settings)
	if err != nil {
		return r, err
	}

	trial.SetName(msg.Context(), trialName)
	trial.SetDescription(msg.Context(), settingMap["description"])
	trial.SetState(msg.Context(), storage.TrialStateOpen)

	if v, ok := settingMap["announcechannel"]; !ok {
		trial.SetAnnounceChannel(msg.Context(), gsettings.AnnounceChannel)
	} else {
		trial.SetAnnounceChannel(msg.Context(), v)
	}

	if v, ok := settingMap["announceto"]; ok {
		trial.SetAnnounceTo(msg.Context(), v)
	}

	if v, ok := settingMap["signupchannel"]; !ok {
		trial.SetSignupChannel(msg.Context(), gsettings.SignupChannel)
	} else {
		trial.SetSignupChannel(msg.Context(), v)
	}

	roleCtEmoList, err := parseRolesString(settingMap["roles"])
	if err != nil {
		return r, err
	}
	for _, rce := range roleCtEmoList {
		if rce.ct != 0 {
			trial.SetRoleCount(msg.Context(), rce.role, rce.emo, rce.ct)
		}
	}

	if err = t.SaveTrial(msg.Context(), trial); err != nil {
		return r, errors.Wrap(err, "could not save event")
	}

	if err = t.Commit(msg.Context()); err != nil {
		return r, errors.Wrap(err, "could not save event")
	}

	level.Info(logger).Message("trial created", "trial_name", trialName)
	r.Description = fmt.Sprintf("Event %q created successfully", trialName)

	return r, nil
}
