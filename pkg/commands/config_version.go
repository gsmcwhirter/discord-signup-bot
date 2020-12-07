package commands

import (
	"github.com/gsmcwhirter/discord-bot-lib/v17/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v17/logging"
	"github.com/gsmcwhirter/go-util/v7/logging/level"
)

func (c *configCommands) version(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "configCommands.version", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To:          cmdhandler.UserMentionString(msg.UserID()),
		Description: c.versionStr,
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling configCommand", "command", "version")

	return r, msg.ContentErr()
}
