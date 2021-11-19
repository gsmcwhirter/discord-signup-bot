package commands

import (
	"context"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
	"github.com/gsmcwhirter/discord-bot-lib/v23/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
)

func (c *ConfigCommands) resetInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "configCommands.reset", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling config interaction", "command", "reset")

	// No colors here because this is a debug mechanism that we want to reduce error-cases in
	// Also no admin channel checks in case we are trying to figure out why the admin channel is broken

	if err := c.reset(ctx, ix.GuildID()); err != nil {
		return r, nil, err
	}

	return c.listInteraction(ix, opts)
}

func (c *ConfigCommands) resetHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "configCommands.reset", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling configCommand", "command", "reset")

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	// No colors here because this is a debug mechanism that we want to reduce error-cases in
	// Also no admin channel checks in case we are trying to figure out why the admin channel is broken

	if err := c.reset(ctx, msg.GuildID()); err != nil {
		return r, err
	}

	return c.listHandler(msg)
}

func (c *ConfigCommands) reset(ctx context.Context, gid snowflake.Snowflake) error {
	ctx, span := c.deps.Census().StartSpan(ctx, "configCommands.reset", "guild_id", gid.ToString())
	defer span.End()

	t, err := c.deps.GuildAPI().NewTransaction(ctx, true)
	if err != nil {
		return err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	bGuild, err := t.AddGuild(ctx, gid.ToString())
	if err != nil {
		return errors.Wrap(err, "unable to find or add guild")
	}

	s := storage.GuildSettings{}
	bGuild.SetSettings(ctx, s)

	err = t.SaveGuild(ctx, bGuild)
	if err != nil {
		return errors.Wrap(err, "could not save guild settings")
	}

	err = t.Commit(ctx)
	if err != nil {
		return errors.Wrap(err, "could not save guild settings")
	}

	return errors.Wrap(c.deps.PermissionsManager().RefreshPermissions(ctx, c.deps.Bot().Config().ClientID, gid), "could not refresh command permissions")
}
