package logging

import (
	"github.com/go-kit/kit/log"
	"github.com/gsmcwhirter/discord-bot-lib/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/logging"
)

// WithMessage wraps a logger with fields from a cmdhandler.Message
func WithMessage(msg cmdhandler.Message, logger log.Logger) log.Logger {
	logger = logging.WithContext(msg.Context(), logger)
	logger = log.With(logger, "user_id", msg.UserID().ToString(), "channel_id", msg.ChannelID().ToString(), "guild_id", msg.GuildID().ToString())
	return logger
}
