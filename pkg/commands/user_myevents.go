package commands

import (
	"context"
	"fmt"
	"sort"
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

func (c *UserCommands) myEventsInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "userCommands.myEventsInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.EmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling root interaction", "command", "myEvents")

	gsettings, err := storage.GetSettings(ctx, c.deps.GuildAPI(), ix.GuildID())
	if err != nil {
		return r, nil, err
	}

	okColor, err := colorToInt(gsettings.MessageColor)
	if err != nil {
		return r, nil, err
	}

	errColor, err := colorToInt(gsettings.ErrorColor)
	if err != nil {
		return r, nil, err
	}

	r.SetColor(errColor)

	tNames, err := c.myEvents(ctx, ix.GuildID(), ix.UserID())
	if err != nil {
		return r, nil, errors.Wrap(err, "could not retrieve user events")
	}

	var listContent string
	if len(tNames) > 0 {
		listContent = strings.Join(tNames, "\n")
	} else {
		listContent = "(none yet)"
	}

	r.Fields = []cmdhandler.EmbedField{
		{
			Name: "*Registered Events*",
			Val:  listContent,
		},
	}

	r.SetColor(okColor)
	r.SetEphemeral(true)

	return r, nil, nil
}

func (c *UserCommands) myEventsHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "userCommands.myEventsHandler", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.EmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling rootCommand", "command", "myEvents")

	gsettings, err := storage.GetSettings(ctx, c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	okColor, err := colorToInt(gsettings.MessageColor)
	if err != nil {
		return r, err
	}

	errColor, err := colorToInt(gsettings.ErrorColor)
	if err != nil {
		return r, err
	}

	r.SetColor(errColor)

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	tNames, err := c.myEvents(ctx, msg.GuildID(), msg.UserID())
	if err != nil {
		return r, errors.Wrap(err, "could not retrieve user events")
	}

	var listContent string
	if len(tNames) > 0 {
		listContent = strings.Join(tNames, "\n")
	} else {
		listContent = "(none yet)"
	}

	r.Fields = []cmdhandler.EmbedField{
		{
			Name: "*Registered Events*",
			Val:  listContent,
		},
	}

	r.SetColor(okColor)

	return r, nil
}

func (c *UserCommands) myEvents(ctx context.Context, gid, uid snowflake.Snowflake) ([]string, error) {
	ctx, span := c.deps.Census().StartSpan(ctx, "userCommands.myEvents", "guild_id", gid.ToString())
	defer span.End()

	t, err := c.deps.TrialAPI().NewTransaction(ctx, gid.ToString(), false)
	if err != nil {
		return nil, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	g, ok := c.deps.BotSession().Guild(gid)
	if !ok {
		return nil, ErrGuildNotFound
	}

	trials := t.GetTrials(ctx)
	tNames := make([]string, 0, len(trials))
	for _, trial := range trials {
		if trial.GetState(ctx) == storage.TrialStateClosed {
			continue
		}

		signups := trial.GetSignups(ctx)
		signedUp := false
		role := ""
		for _, su := range signups {
			if su.GetName(ctx) == cmdhandler.UserMentionString(uid) {
				signedUp = true
				role = su.GetRole(ctx)
				break
			}
		}

		if !signedUp {
			continue
		}

		if tscID, ok := g.ChannelWithName(trial.GetSignupChannel(ctx)); ok {
			tNames = append(tNames, fmt.Sprintf("%s as %s (%s)", trial.GetName(ctx), role, cmdhandler.ChannelMentionString(tscID)))
		} else {
			tNames = append(tNames, fmt.Sprintf("%s as %s", trial.GetName(ctx), role))
		}
	}
	sort.Strings(tNames)

	return tNames, nil
}
