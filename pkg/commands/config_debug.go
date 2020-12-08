package commands

import (
	"fmt"

	"github.com/gsmcwhirter/go-util/v7/deferutil"
	"github.com/gsmcwhirter/go-util/v7/errors"
	"github.com/gsmcwhirter/go-util/v7/logging/level"

	"github.com/gsmcwhirter/discord-bot-lib/v18/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v18/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v18/snowflake"
)

func (c *configCommands) debug(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "configCommands.list", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling configCommand", "command", "list")

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	t, err := c.deps.GuildAPI().NewTransaction(msg.Context(), false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(msg.Context()) })

	bGuild, err := t.AddGuild(msg.Context(), msg.GuildID().ToString())
	if err != nil {
		return r, errors.Wrap(err, "unable to find guild")
	}

	s := bGuild.GetSettings(msg.Context())

	var adminChannelID, announceChannelID, signupChannelID snowflake.Snowflake

	g, ok := c.deps.BotSession().Guild(msg.GuildID())
	if !ok {
		return r, errors.New("could not find guild in session")
	}

	if s.AdminChannel != "" {
		if cid, ok := g.ChannelWithName(s.AdminChannel); ok {
			adminChannelID = cid
		}
	}

	if s.AnnounceChannel != "" {
		if cid, ok := g.ChannelWithName(s.AnnounceChannel); ok {
			announceChannelID = cid
		}
	}

	if s.SignupChannel != "" {
		if cid, ok := g.ChannelWithName(s.SignupChannel); ok {
			signupChannelID = cid
		}
	}

	dbgString := fmt.Sprintf(`
GuildSettings:
	- Guild ID: %[1]s,
	- Your ID: %[10]s,
	- Current Channel ID: %[14]s,

	- ControlSequence: '%[2]s',
	- AnnounceTo: '%[6]s', 
	- ShowAfterSignup: '%[7]s',
	- ShowAfterWithdraw: '%[8]s',
	
	- AnnounceChannel: '#%[3]s',
	- AnnounceChannel ID: %[11]s,

	- SignupChannel: '#%[4]s',
	- SignupChannel ID: %[12]s,

	- AdminChannel: '#%[5]s',
	- AdminChannel ID: %[13]s,

	- AdminRole: '<@&%[9]s>',
	- AdminRole ID: %[9]s,

	`,
		msg.GuildID().ToString(),
		s.ControlSequence,
		s.AnnounceChannel,
		s.SignupChannel,
		s.AdminChannel,
		s.AnnounceTo,
		s.ShowAfterSignup,
		s.ShowAfterWithdraw,
		s.AdminRole,
		msg.UserID().ToString(),
		announceChannelID.ToString(),
		signupChannelID.ToString(),
		adminChannelID.ToString(),
		msg.ChannelID().ToString())

	r.Description = dbgString
	return r, nil
}
