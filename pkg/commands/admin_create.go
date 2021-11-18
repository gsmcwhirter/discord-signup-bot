package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	log "github.com/gsmcwhirter/go-util/v8/logging"
	"github.com/gsmcwhirter/go-util/v8/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
	"github.com/gsmcwhirter/discord-bot-lib/v23/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
)

type eventSettings struct {
	Description           *string
	AnnounceChannel       *string
	SignupChannel         *string
	AnnounceTo            *string
	HideReactionsAnnounce *string
	HideReactionsShow     *string
	Time                  *string
	RoleOrder             *string
	Roles                 *string
}

var (
	trueString  = "true"
	falseString = "false"
)

var ErrMissingData = errors.New("missing data")

func (c *AdminCommands) createInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "adminCommands.createInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling admin interaction", "command", "create")

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

	if err := c.create(ctx, logger, ix.GuildID(), gsettings, eventName, es); err != nil {
		return r, nil, errors.Wrap(err, "could not create event")
	}

	level.Info(logger).Message("trial created", "trial_name", eventName)
	r.Description = fmt.Sprintf("Event %q created successfully", eventName)
	r.SetColor(okColor)

	return r, nil, nil
}

func (c *AdminCommands) createHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "adminCommands.createHandler", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling adminCommand", "command", "create", "args", msg.Contents())

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

	if err := c.create(ctx, logger, msg.GuildID(), gsettings, trialName, es); err != nil {
		return r, errors.Wrap(err, "could not create event")
	}

	level.Info(logger).Message("trial created", "trial_name", trialName)
	r.Description = fmt.Sprintf("Event %q created successfully", trialName)
	r.SetColor(okColor)

	return r, nil
}

func (c *AdminCommands) create(ctx context.Context, logger log.Logger, gid snowflake.Snowflake, gsettings storage.GuildSettings, eventName string, settings eventSettings) error {
	ctx, span := c.deps.Census().StartSpan(ctx, "adminCommands.create", "guild_id", gid.ToString())
	defer span.End()

	level.Debug(logger).Message("event settings", "data", fmt.Sprintf("%#v", settings))

	t, err := c.deps.TrialAPI().NewTransaction(ctx, gid.ToString(), true)
	if err != nil {
		return err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	trial, err := t.AddTrial(ctx, eventName)
	if err != nil {
		return err
	}

	trial.SetName(ctx, eventName)
	trial.SetDescription(ctx, *settings.Description)
	trial.SetState(ctx, storage.TrialStateOpen)

	if settings.AnnounceChannel == nil {
		trial.SetAnnounceChannel(ctx, gsettings.AnnounceChannel)
	} else {
		trial.SetAnnounceChannel(ctx, *settings.AnnounceChannel)
	}

	if settings.AnnounceTo != nil {
		trial.SetAnnounceTo(ctx, *settings.AnnounceTo)
	}

	if settings.SignupChannel == nil {
		trial.SetSignupChannel(ctx, gsettings.SignupChannel)
	} else {
		trial.SetSignupChannel(ctx, *settings.SignupChannel)
	}

	if settings.HideReactionsAnnounce == nil {
		err = trial.SetHideReactionsAnnounce(ctx, gsettings.HideReactionsAnnounce)
	} else {
		err = trial.SetHideReactionsAnnounce(ctx, *settings.HideReactionsAnnounce)
	}
	if err != nil {
		return err
	}

	if settings.HideReactionsShow == nil {
		err = trial.SetHideReactionsShow(ctx, gsettings.HideReactionsShow)
	} else {
		err = trial.SetHideReactionsShow(ctx, *settings.HideReactionsShow)
	}
	if err != nil {
		return err
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

	var roles string
	if settings.Roles == nil {
		roles = ""
	} else {
		roles = *settings.Roles
	}

	roleCtEmoList, err := parseRolesString(roles)
	if err != nil {
		return err
	}
	for _, rce := range roleCtEmoList {
		if rce.ct != 0 {
			trial.SetRoleCount(ctx, rce.role, rce.emo, rce.ct)
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
