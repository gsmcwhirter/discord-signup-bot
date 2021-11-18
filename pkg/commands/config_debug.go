package commands

import (
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/logging/level"

	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
)

func (c *ConfigCommands) debugHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
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

	t, err := c.deps.GuildAPI().NewTransaction(ctx, false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	bGuild, err := t.AddGuild(ctx, msg.GuildID().ToString())
	if err != nil {
		return r, errors.Wrap(err, "unable to find guild")
	}

	s := bGuild.GetSettings(ctx)

	okColor, err := colorToInt(s.MessageColor)
	if err != nil {
		return r, err
	}

	errColor, err := colorToInt(s.ErrorColor)
	if err != nil {
		return r, err
	}

	r.SetColor(errColor)

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

	adminRoles := make([]string, 0, len(s.AdminRoles))
	for _, rname := range s.AdminRoles {
		adminRoles = append(adminRoles, fmt.Sprintf("<@&%s>", rname))
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
	- MessageColor: '%[16]s',
	- ErrorColor: '%[17]s',
	
	- AnnounceChannel: '#%[3]s',
	- AnnounceChannel ID: %[11]s,

	- SignupChannel: '#%[4]s',
	- SignupChannel ID: %[12]s,

	- AdminChannel: '#%[5]s',
	- AdminChannel ID: %[13]s,

	- AdminRole: '%[9]s',
	- AdminRole IDs: %[15]s,

	`,
		msg.GuildID().ToString(),
		s.ControlSequence,
		s.AnnounceChannel,
		s.SignupChannel,
		s.AdminChannel,
		s.AnnounceTo,
		s.ShowAfterSignup,
		s.ShowAfterWithdraw,
		strings.Join(adminRoles, ", "),
		msg.UserID().ToString(),
		announceChannelID.ToString(),
		signupChannelID.ToString(),
		adminChannelID.ToString(),
		msg.ChannelID().ToString(),
		s.AdminRoles,
		s.MessageColor,
		s.ErrorColor,
	)

	r.Description = dbgString
	r.Color = okColor
	return r, nil
}
