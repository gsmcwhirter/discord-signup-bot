package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
	"github.com/gsmcwhirter/discord-bot-lib/v23/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
)

func (c *AdminCommands) editInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "adminCommands.editInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling admin interaction", "command", "edit")

	gsettings, err := storage.GetSettings(ctx, c.deps.GuildAPI(), ix.GuildID())
	if err != nil {
		return r, nil, err
	}

	okColor, err := colorToInt(gsettings.MessageColor)
	if err != nil {
		return r, nil, err
	}

	errColor, err := colorToInt(gsettings.ErrorColor)
	if err != nil {
		return r, nil, err
	}

	r.SetColor(errColor)

	if !isAdminChannel(logger, ix, gsettings.AdminChannel, c.deps.BotSession()) {
		level.Info(logger).Message("command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return nil, nil, msghandler.ErrUnauthorized
	}

	eventName, es, err := eventSettingsFromOptions(opts, ix.Data.Resolved)
	if err != nil {
		return r, nil, errors.Wrap(err, "could not parse interaction data")
	}

	if err := c.edit(ctx, ix.GuildID(), eventName, es); err != nil {
		return r, nil, errors.Wrap(err, "could not edit event")
	}

	level.Info(logger).Message("trial edited", "trial_name", eventName)
	r.Description = fmt.Sprintf("Trial %s edited successfully", eventName)
	r.SetColor(okColor)

	return r, nil, nil
}

func (c *AdminCommands) editHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "adminCommands.editHandler", "guild_id", msg.GuildID().ToString())
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

	okColor, err := colorToInt(gsettings.MessageColor)
	if err != nil {
		return r, err
	}

	errColor, err := colorToInt(gsettings.ErrorColor)
	if err != nil {
		return r, err
	}

	r.SetColor(errColor)

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

	settingMap, err := parseSettingDescriptionArgs(settings)
	if err != nil {
		return r, err
	}
	es := loadEventSettings(settingMap)

	if err := c.edit(ctx, msg.GuildID(), trialName, es); err != nil {
		return r, errors.Wrap(err, "could not edit event")
	}

	level.Info(logger).Message("trial edited", "trial_name", trialName)
	r.Description = fmt.Sprintf("Trial %s edited successfully", trialName)
	r.SetColor(okColor)

	return r, nil
}

func (c *AdminCommands) edit(ctx context.Context, gid snowflake.Snowflake, eventName string, settings eventSettings) error {
	t, err := c.deps.TrialAPI().NewTransaction(ctx, gid.ToString(), true)
	if err != nil {
		return err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	trial, err := t.GetTrial(ctx, eventName)
	if err != nil {
		return err
	}

	if settings.Description != nil {
		trial.SetDescription(ctx, *settings.Description)
	}

	if settings.AnnounceChannel != nil {
		trial.SetAnnounceChannel(ctx, *settings.AnnounceChannel)
	}

	if settings.AnnounceTo != nil {
		trial.SetAnnounceTo(ctx, *settings.AnnounceTo)
	}

	if settings.SignupChannel != nil {
		trial.SetSignupChannel(ctx, *settings.SignupChannel)
	}

	if settings.HideReactionsAnnounce != nil {
		if err := trial.SetHideReactionsAnnounce(ctx, *settings.HideReactionsAnnounce); err != nil {
			return err
		}
	}

	if settings.HideReactionsShow != nil {
		if err := trial.SetHideReactionsShow(ctx, *settings.HideReactionsShow); err != nil {
			return err
		}
	}

	if settings.Time != nil {
		trial.SetTime(ctx, *settings.Time)
	}

	if settings.RoleOrder != nil {
		roleOrder := strings.Split(*settings.RoleOrder, ",")
		for i := range roleOrder {
			roleOrder[i] = strings.TrimSpace(roleOrder[i])
		}
		trial.SetRoleOrder(ctx, roleOrder)
	}

	if settings.Roles != nil {
		roleCtEmoList, err := parseRolesString(*settings.Roles)
		if err != nil {
			return err
		}
		for _, rce := range roleCtEmoList {
			if rce.ct == 0 {
				trial.RemoveRole(ctx, rce.role)
			} else {
				trial.SetRoleCount(ctx, rce.role, rce.emo, rce.ct)
			}
		}
	}

	if err = t.SaveTrial(ctx, trial); err != nil {
		return errors.Wrap(err, "could not save event")
	}

	if err = t.Commit(ctx); err != nil {
		return errors.Wrap(err, "could not save event")
	}

	return nil
}
