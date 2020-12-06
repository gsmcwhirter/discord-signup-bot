package msghandler

import (
	"context"

	"github.com/gsmcwhirter/discord-bot-lib/v16/bot"
	"github.com/gsmcwhirter/discord-bot-lib/v16/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v16/etfapi"
	"github.com/gsmcwhirter/discord-bot-lib/v16/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v16/snowflake"
	"github.com/gsmcwhirter/go-util/v7/logging/level"
)

// IsAdminAuthorized determines if a user can take admin actions with the bot (ignoring channel)
func IsAdminAuthorized(ctx context.Context, logger logging.Logger, msg cmdhandler.Message, adminRole string, session *etfapi.Session, b bot.DiscordBot) bool {
	authorized := false
	authorized = authorized || session.IsGuildAdmin(msg.GuildID(), msg.UserID())
	authorized = authorized || HasAdminRole(ctx, logger, msg, adminRole, b)

	return authorized
}

// IsAdminChannel determines if a message is occurring in the admin channel for a guild
func IsAdminChannel(logger logging.Logger, msg cmdhandler.Message, adminChannel string, session *etfapi.Session) bool {

	g, ok := session.Guild(msg.GuildID())
	if !ok {
		level.Error(logger).Message("could not find guild in session")
		return false
	}

	if adminChannel == "" {
		return true
	}

	cid, ok := g.ChannelWithName(adminChannel)
	if !ok {
		return false
	}

	return cid == msg.ChannelID()
}

// IsSignupChannel determines if a message is occurring in the designated signup channel for a guild
func IsSignupChannel(msg cmdhandler.Message, signupChannel string, session *etfapi.Session) bool {
	g, ok := session.Guild(msg.GuildID())
	if !ok {
		return false
	}

	cid, ok := g.ChannelWithName(signupChannel)
	if !ok {
		return false
	}

	return cid == msg.ChannelID()
}

// HasAdminRole determines if the message author is an authorized bot admin (not super-admin)
func HasAdminRole(ctx context.Context, logger logging.Logger, msg cmdhandler.Message, adminRole string, b bot.DiscordBot) bool {
	if adminRole == "" {
		return false
	}

	rid, err := snowflake.FromString(adminRole)
	if err != nil {
		level.Error(logger).Err("could not parse AdminRole", err, "admin_role", adminRole)
		return false
	}

	// g, ok := session.Guild(msg.GuildID())
	// if !ok {
	// 	level.Error(logger).Message("could not find guild in session")
	// 	return false
	// }

	// return g.HasRole(msg.UserID(), rid)

	gm, err := b.GetGuildMember(ctx, msg.GuildID(), msg.UserID())
	if err != nil {
		level.Error(logger).Err("could not get guild member", err, "guild_id", msg.GuildID(), "member_id", msg.UserID())
		return false
	}
	return gm.HasRole(rid)
}
