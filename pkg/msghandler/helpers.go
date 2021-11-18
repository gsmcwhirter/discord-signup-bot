package msghandler

import (
	"context"

	"github.com/gsmcwhirter/discord-bot-lib/v23/bot"
	"github.com/gsmcwhirter/discord-bot-lib/v23/bot/session"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
	"github.com/gsmcwhirter/go-util/v8/logging/level"
)

type Logger = interface {
	Log(keyvals ...interface{}) error
	Message(string, ...interface{})
	Err(string, error, ...interface{})
	Printf(string, ...interface{})
}

type MessageLike interface {
	UserID() snowflake.Snowflake
	GuildID() snowflake.Snowflake
	ChannelID() snowflake.Snowflake
}

// IsAdminAuthorized determines if a user can take admin actions with the bot (ignoring channel)
func IsAdminAuthorized(ctx context.Context, logger Logger, msg MessageLike, adminRoles []string, sess *session.Session, b *bot.DiscordBot) bool {
	authorized := false
	authorized = authorized || sess.IsGuildAdmin(msg.GuildID(), msg.UserID())
	authorized = authorized || HasAdminRole(ctx, logger, sess, msg, adminRoles, b)

	return authorized
}

// IsAdminChannel determines if a message is occurring in the admin channel for a guild
func IsAdminChannel(logger Logger, msg MessageLike, adminChannel string, sess *session.Session) bool {
	g, ok := sess.Guild(msg.GuildID())
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
func IsSignupChannel(msg MessageLike, signupChannel string, sess *session.Session) bool {
	g, ok := sess.Guild(msg.GuildID())
	if !ok {
		return false
	}

	cid, ok := g.ChannelWithName(signupChannel)
	if !ok {
		return false
	}

	return cid == msg.ChannelID()
}

func hasAdminRole(ctx context.Context, logger Logger, gm entity.GuildMember, role string) bool {
	rid, err := snowflake.FromString(role)
	if err != nil {
		level.Error(logger).Err("could not parse AdminRole", err, "admin_role", role)
		return false
	}

	return gm.HasRole(rid)
}

func hasRoleWithAdministrator(ctx context.Context, logger Logger, sess *session.Session, gid snowflake.Snowflake, gm entity.GuildMember) bool {
	g, ok := sess.Guild(gid)
	if !ok {
		level.Error(logger).Message("could not find guild in the session", "guild_id", gid)
		return false
	}

	for _, rid := range gm.RoleSnowflakes {
		if g.RoleIsAdministrator(rid) {
			return true
		}
	}

	return false
}

// HasAdminRole determines if the message author is an authorized bot admin (not super-admin)
func HasAdminRole(ctx context.Context, logger Logger, sess *session.Session, msg MessageLike, adminRoles []string, b *bot.DiscordBot) bool {
	gm, err := b.API().GetGuildMember(ctx, msg.GuildID(), msg.UserID())
	if err != nil {
		level.Error(logger).Err("could not get guild member", err, "guild_id", msg.GuildID(), "member_id", msg.UserID())
		return false
	}

	if hasRoleWithAdministrator(ctx, logger, sess, msg.GuildID(), gm) {
		return true
	}

	if len(adminRoles) == 0 {
		return false
	}

	for _, role := range adminRoles {
		if hasAdminRole(ctx, logger, gm, role) {
			return true
		}
	}

	return false
}
