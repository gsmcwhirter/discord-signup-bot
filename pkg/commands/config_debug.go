package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
	"github.com/gsmcwhirter/discord-bot-lib/v23/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
)

func (c *ConfigCommands) debugInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "configCommands.debugInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling admin interaction", "command", "announce")

	gsettings, err := storage.GetSettings(ctx, c.deps.GuildAPI(), ix.GuildID())
	if err != nil {
		return r, nil, err
	}

	// No color here to eliminate another class of errors, since this is debug
	// Also no admin channel checks in case we are trying to figure out why the admin channel is broken

	r2, err := c.debug(ctx, ix.GuildID(), ix.ChannelID(), ix.UserID(), gsettings)
	if err != nil {
		return r, nil, errors.Wrap(err, "could not debug config")
	}

	return r2, nil, nil
}

func (c *ConfigCommands) debugHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "configCommands.debugHandler", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling configCommand", "command", "debug")

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	gsettings, err := storage.GetSettings(ctx, c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	// No color here to eliminate another class of errors, since this is debug
	// Also no admin channel checks in case we are trying to figure out why the admin channel is broken

	r2, err := c.debug(ctx, msg.GuildID(), msg.ChannelID(), msg.UserID(), gsettings)
	if err != nil {
		return r, errors.Wrap(err, "could not debug config")
	}
	r2.SetReplyTo(msg)

	return r2, nil
}

func (c *ConfigCommands) debug(ctx context.Context, gid, currCid, uid snowflake.Snowflake, gsettings storage.GuildSettings) (*cmdhandler.SimpleEmbedResponse, error) {
	ctx, span := c.deps.Census().StartSpan(ctx, "configCommands.debug", "guild_id", gid.ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	t, err := c.deps.GuildAPI().NewTransaction(ctx, false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	var adminChannelID, announceChannelID, signupChannelID snowflake.Snowflake

	g, ok := c.deps.BotSession().Guild(gid)
	if !ok {
		return r, errors.New("could not find guild in session")
	}

	if gsettings.AdminChannel != "" {
		if cid, ok := g.ChannelWithName(gsettings.AdminChannel); ok {
			adminChannelID = cid
		}
	}

	if gsettings.AnnounceChannel != "" {
		if cid, ok := g.ChannelWithName(gsettings.AnnounceChannel); ok {
			announceChannelID = cid
		}
	}

	if gsettings.SignupChannel != "" {
		if cid, ok := g.ChannelWithName(gsettings.SignupChannel); ok {
			signupChannelID = cid
		}
	}

	adminRoles := make([]string, 0, len(gsettings.AdminRoles))
	for _, rname := range gsettings.AdminRoles {
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
		gid.ToString(),
		gsettings.ControlSequence,
		gsettings.AnnounceChannel,
		gsettings.SignupChannel,
		gsettings.AdminChannel,
		gsettings.AnnounceTo,
		gsettings.ShowAfterSignup,
		gsettings.ShowAfterWithdraw,
		strings.Join(adminRoles, ", "),
		uid.ToString(),
		announceChannelID.ToString(),
		signupChannelID.ToString(),
		adminChannelID.ToString(),
		currCid.ToString(),
		gsettings.AdminRoles,
		gsettings.MessageColor,
		gsettings.ErrorColor,
	)

	r.Description = dbgString

	return r, nil
}
