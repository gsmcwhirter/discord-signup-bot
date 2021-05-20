package reactions

import (
	log "github.com/gsmcwhirter/go-util/v8/logging"
)

// LoggerWithReaction wraps a logger with fields from a reactions.Reaction
func LoggerWithReaction(r Reaction, logger log.Logger) log.Logger {
	logger = log.WithContext(r.Context(), logger)
	logger = log.With(logger, "user_id", r.UserID().ToString(), "channel_id", r.ChannelID().ToString(), "guild_id", r.GuildID().ToString(), "message_id", r.MessageID().ToString(), "emoji", r.Emoji())
	return logger
}
