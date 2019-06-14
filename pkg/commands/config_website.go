package commands

import (
	"github.com/gsmcwhirter/discord-bot-lib/v7/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v7/logging"
	"github.com/gsmcwhirter/go-util/v3/logging/level"
)

func (c *configCommands) website(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To:          cmdhandler.UserMentionString(msg.UserID()),
		Description: "https://www.evogames.org/bots/eso-signup-bot/",
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling configCommand", "command", "website")

	return r, msg.ContentErr()
}
