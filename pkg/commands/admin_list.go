package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
	"github.com/gsmcwhirter/discord-bot-lib/v23/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
)

func (c *AdminCommands) listInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "adminCommands.listInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.EmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling admin interaction", "command", "list")

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

	if !isAdminChannel(logger, ix, gsettings.AdminChannel, c.deps.BotSession()) {
		level.Info(logger).Message("command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, nil, msghandler.ErrUnauthorized
	}

	tNamesOpen, tNamesClosed, err := c.list(ctx, ix.GuildID())
	if err != nil {
		return r, nil, errors.Wrap(err, "could not produce event lists")
	}

	r.To = "Event List"
	r.Fields = []cmdhandler.EmbedField{
		{
			Name: "*Available Events*",
			Val:  fmt.Sprintf("```\n%s\n```\n", strings.Join(tNamesOpen, "\n")),
		},
		{
			Name: "*Closed Events*",
			Val:  fmt.Sprintf("```\n%s\n```\n", strings.Join(tNamesClosed, "\n")),
		},
	}
	r.SetColor(okColor)

	return r, nil, nil
}

func (c *AdminCommands) listHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "adminCommands.listHandler", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.EmbedResponse{}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling adminCommand", "command", "list")

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

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		level.Info(logger).Message("command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, msghandler.ErrUnauthorized
	}

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	tNamesOpen, tNamesClosed, err := c.list(ctx, msg.GuildID())
	if err != nil {
		return r, errors.Wrap(err, "could not produce event lists")
	}

	r.Fields = []cmdhandler.EmbedField{
		{
			Name: "*Available Events*",
			Val:  fmt.Sprintf("```\n%s\n```\n", strings.Join(tNamesOpen, "\n")),
		},
		{
			Name: "*Closed Events*",
			Val:  fmt.Sprintf("```\n%s\n```\n", strings.Join(tNamesClosed, "\n")),
		},
	}
	r.SetColor(okColor)

	return r, nil
}

func (c *AdminCommands) list(ctx context.Context, gid snowflake.Snowflake) (open, closed []string, err error) {
	t, err := c.deps.TrialAPI().NewTransaction(ctx, gid.ToString(), false)
	if err != nil {
		return nil, nil, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	trials := t.GetTrials(ctx)
	tNamesOpen := make([]string, 0, len(trials))
	tNamesClosed := make([]string, 0, len(trials))
	for _, trial := range trials {
		if trial.GetState(ctx) == storage.TrialStateClosed {
			tNamesClosed = append(tNamesClosed, fmt.Sprintf("%s (#%s)", trial.GetName(ctx), trial.GetSignupChannel(ctx)))
		} else {
			tNamesOpen = append(tNamesOpen, fmt.Sprintf("%s (#%s)", trial.GetName(ctx), trial.GetSignupChannel(ctx)))
		}
	}
	sort.Strings(tNamesOpen)
	sort.Strings(tNamesClosed)

	return tNamesOpen, tNamesClosed, nil
}
