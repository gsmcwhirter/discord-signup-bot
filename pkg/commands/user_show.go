package commands

import (
	"context"
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

func (c *UserCommands) showInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "userCommands.showInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling root interaction", "command", "show")

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

	var eventName string
	for i := range opts {
		if opts[i].Name == "event_name" {
			eventName = opts[i].ValueString
			continue
		}
	}

	_, r2, err := c.show(ctx, ix.GuildID(), eventName)
	if err != nil {
		return r, nil, errors.Wrap(err, "could not find trial")
	}

	r2.SetColor(okColor)
	r2.SetEphemeral(true)

	return r2, nil, nil
}

func (c *UserCommands) showHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "userCommands.showHandler", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling rootCommand", "command", "show", "trial_name", msg.Contents())

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

	if len(msg.Contents()) != 1 {
		return r, errors.New("you must supply exactly 1 argument -- trial name; are you missing quotes?")
	}

	trialName := strings.TrimSpace(msg.Contents()[0])

	trial, r2, err := c.show(ctx, msg.GuildID(), trialName)
	if err != nil {
		return r, errors.Wrap(err, "could not find trial")
	}

	if !isSignupChannel(ctx, logger, msg, trial.GetSignupChannel(ctx), gsettings.AdminChannel, gsettings.AdminRoles, c.deps.BotSession(), c.deps.Bot()) {
		level.Info(logger).Message("command not in signup channel", "signup_channel", trial.GetSignupChannel(ctx))
		return r, msghandler.ErrNoResponse
	}

	r2.SetReplyTo(msg)
	r2.SetColor(okColor)

	return r2, nil
}

func (c *UserCommands) show(ctx context.Context, gid snowflake.Snowflake, eventName string) (storage.Trial, *cmdhandler.EmbedResponse, error) {
	t, err := c.deps.TrialAPI().NewTransaction(ctx, gid.ToString(), false)
	if err != nil {
		return nil, nil, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	trial, err := t.GetTrial(ctx, eventName)
	if err != nil {
		return nil, nil, err
	}

	r2 := formatTrialDisplay(ctx, trial, true)

	return trial, r2, nil
}
