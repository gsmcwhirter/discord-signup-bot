package msghandler

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gsmcwhirter/discord-bot-lib/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/etfapi"
	"github.com/gsmcwhirter/discord-bot-lib/snowflake"
)

// IsAdminAuthorized determines if a user can take admin actions with the bot (ignoring channel)
func IsAdminAuthorized(logger log.Logger, msg cmdhandler.Message, adminRole string, session *etfapi.Session) bool {
	authorized := false
	authorized = authorized || session.IsGuildAdmin(msg.GuildID(), msg.UserID())
	authorized = authorized || HasAdminRole(logger, msg, adminRole, session)

	return authorized
}

// IsAdminChannel determines if a message is occurring in the admin channel for a guild
func IsAdminChannel(logger log.Logger, msg cmdhandler.Message, adminChannel string, session *etfapi.Session) bool {

	g, ok := session.Guild(msg.GuildID())
	if !ok {
		_ = level.Error(logger).Log("message", "could not find guild in session")
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
func HasAdminRole(logger log.Logger, msg cmdhandler.Message, adminRole string, session *etfapi.Session) bool {
	if adminRole == "" {
		return false
	}

	rid, err := snowflake.FromString(adminRole)
	if err != nil {
		_ = level.Error(logger).Log("message", "could not parse AdminRole", "admin_role", adminRole, "err", err)
		return false
	}

	g, ok := session.Guild(msg.GuildID())
	if !ok {
		_ = level.Error(logger).Log("message", "could not find guild in session")
		return false
	}

	return g.HasRole(msg.UserID(), rid)
}
