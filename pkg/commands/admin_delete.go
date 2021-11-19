package commands

import (
	"context"
	"fmt"

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

func (c *AdminCommands) deleteInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "adminCommands.deleteInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling admin interaction", "command", "delete")

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
		return r, nil, msghandler.ErrUnauthorized
	}

	var eventName string
	for i := range opts {
		if opts[i].Name == "event_name" {
			eventName = opts[i].ValueString
			continue
		}
	}

	if err := c.delete(ctx, ix.GuildID(), eventName); err != nil {
		return r, nil, errors.Wrap(err, "could not delete event")
	}

	level.Info(logger).Message("trial deleted", "trial_name", eventName)
	r.Description = fmt.Sprintf("Event %q deleted successfully", eventName)
	r.SetColor(okColor)

	return r, nil, nil
}

func (c *AdminCommands) deleteHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "adminCommands.deleteHandler", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling adminCommand", "command", "delete", "args", msg.Contents())

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
		return r, msghandler.ErrUnauthorized
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

	if err := c.delete(ctx, msg.GuildID(), trialName); err != nil {
		return r, errors.Wrap(err, "could not delete event")
	}

	level.Info(logger).Message("trial deleted", "trial_name", trialName)
	r.Description = fmt.Sprintf("Deleted event %q", trialName)
	r.SetColor(okColor)

	return r, nil
}

func (c *AdminCommands) delete(ctx context.Context, gid snowflake.Snowflake, eventName string) error {
	ctx, span := c.deps.Census().StartSpan(ctx, "adminCommands.delete", "guild_id", gid.ToString())
	defer span.End()

	t, err := c.deps.TrialAPI().NewTransaction(ctx, gid.ToString(), true)
	if err != nil {
		return err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	if err = t.DeleteTrial(ctx, eventName); err != nil {
		return errors.Wrap(err, "could not delete event")
	}

	if err = t.Commit(ctx); err != nil {
		return errors.Wrap(err, "could not delete event")
	}

	return nil
}
