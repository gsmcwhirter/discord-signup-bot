package commands

import (
	"github.com/go-kit/kit/log/level"

	"github.com/gsmcwhirter/discord-bot-lib/v6/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v6/logging"
)

func (c *configCommands) discord(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To:          cmdhandler.UserMentionString(msg.UserID()),
		Description: "https://discord.gg/BgkvvbN",
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling configCommand", "command", "discord")

	return r, msg.ContentErr()
}
