package commands

import (
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v19/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v19/logging"
)

func (c *adminCommands) edit(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "adminCommands.edit", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling adminCommand", "command", "edit", "args", msg.Contents())

	gsettings, err := storage.GetSettings(ctx, c.deps.GuildAPI(), msg.GuildID())
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

	t, err := c.deps.TrialAPI().NewTransaction(ctx, msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	trial, err := t.GetTrial(ctx, trialName)
	if err != nil {
		return r, err
	}

	settingMap, err := parseSettingDescriptionArgs(settings)
	if err != nil {
		return r, err
	}

	if v, ok := settingMap["description"]; ok {
		trial.SetDescription(ctx, v)
	}

	if v, ok := settingMap["announcechannel"]; ok {
		trial.SetAnnounceChannel(ctx, v)
	}

	if v, ok := settingMap["announceto"]; ok {
		trial.SetAnnounceTo(ctx, v)
	}

	if v, ok := settingMap["signupchannel"]; ok {
		trial.SetSignupChannel(ctx, v)
	}

	if v, ok := settingMap["hidereactionsannounce"]; !ok {
		err = trial.SetHideReactionsAnnounce(ctx, gsettings.HideReactionsAnnounce)
	} else {
		err = trial.SetHideReactionsAnnounce(ctx, v)
	}
	if err != nil {
		return r, err
	}

	if v, ok := settingMap["hidereactionsshow"]; !ok {
		err = trial.SetHideReactionsShow(ctx, gsettings.HideReactionsShow)
	} else {
		err = trial.SetHideReactionsShow(ctx, v)
	}
	if err != nil {
		return r, err
	}

	if v, ok := settingMap["roleorder"]; ok {
		roleOrder := strings.Split(v, ",")
		for i := range roleOrder {
			roleOrder[i] = strings.TrimSpace(roleOrder[i])
		}
		trial.SetRoleOrder(ctx, roleOrder)
	}

	roleCtEmoList, err := parseRolesString(settingMap["roles"])
	if err != nil {
		return r, err
	}
	for _, rce := range roleCtEmoList {
		if rce.ct == 0 {
			trial.RemoveRole(ctx, rce.role)
		} else {
			trial.SetRoleCount(ctx, rce.role, rce.emo, rce.ct)
		}
	}

	if err = t.SaveTrial(ctx, trial); err != nil {
		return r, errors.Wrap(err, "could not save event")
	}

	if err = t.Commit(ctx); err != nil {
		return r, errors.Wrap(err, "could not save event")
	}

	level.Info(logger).Message("trial edited", "trial_name", trialName)
	r.Description = fmt.Sprintf("Trial %s edited successfully", trialName)

	return r, nil
}
