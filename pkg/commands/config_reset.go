package commands

import (
	"github.com/gsmcwhirter/go-util/v7/deferutil"
	"github.com/gsmcwhirter/go-util/v7/errors"
	"github.com/gsmcwhirter/go-util/v7/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v13/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v13/logging"
)

func (c *configCommands) reset(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "configCommands.reset", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling configCommand", "command", "reset")

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	t, err := c.deps.GuildAPI().NewTransaction(msg.Context(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(msg.Context()) })

	bGuild, err := t.AddGuild(msg.Context(), msg.GuildID().ToString())
	if err != nil {
		return r, errors.Wrap(err, "unable to find or add guild")
	}

	s := storage.GuildSettings{}
	bGuild.SetSettings(msg.Context(), s)

	err = t.SaveGuild(msg.Context(), bGuild)
	if err != nil {
		return r, errors.Wrap(err, "could not save guild settings")
	}

	err = t.Commit(msg.Context())
	if err != nil {
		return r, errors.Wrap(err, "could not save guild settings")
	}

	return c.list(msg)
}
