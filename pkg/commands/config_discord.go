package commands

import (
	"github.com/gsmcwhirter/discord-bot-lib/v11/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v11/logging"
	"github.com/gsmcwhirter/go-util/v5/logging/level"
)

func (c *configCommands) discord(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "configCommands.discord", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		To:          cmdhandler.UserMentionString(msg.UserID()),
		Description: "https://discord.gg/BgkvvbN",
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling configCommand", "command", "discord")

	return r, msg.ContentErr()
}
