package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	log "github.com/gsmcwhirter/go-util/v8/logging"
	"github.com/gsmcwhirter/go-util/v8/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
	"github.com/gsmcwhirter/discord-bot-lib/v23/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
)

func (c *UserCommands) withdrawInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "userCommands.withdrawInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling root interaction", "command", "withdraw")

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

	r2, err := c.withdraw(ctx, logger, ix, gsettings, false, ix.GuildID(), ix.UserID(), eventName)
	if err != nil {
		return r, nil, errors.Wrap(err, "could not withdraw from event")
	}

	level.Info(logger).Message("withdrew", "trial_name", eventName)

	r.Description = fmt.Sprintf("Withdrew from %s", eventName)
	r.SetColor(okColor)
	r.SetEphemeral(true)

	if r2 != nil {
		r2.SetColor(okColor)
		return r, []cmdhandler.Response{r2}, nil
	}

	return r, nil, nil
}

func (c *UserCommands) withdrawHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "userCommands.withdrawHandler", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling rootCommand", "command", "withdraw", "trial_name", msg.Contents())

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

	if len(msg.Contents()) < 1 {
		return r, errors.New("missing event name")
	}

	trialName := strings.TrimSpace(msg.Contents()[0])

	r2, err := c.withdraw(ctx, logger, msg, gsettings, true, msg.GuildID(), msg.UserID(), trialName)
	if err != nil {
		return r, err // no wrap because of ErrNoResponse
	}

	level.Info(logger).Message("withdrew", "trial_name", trialName)
	descStr := fmt.Sprintf("Withdrew from %s", trialName)

	if r2 != nil {
		r2.Description = fmt.Sprintf("%s\n\n%s", descStr, r2.Description)
		r2.ToChannel = 0
		r2.SetColor(okColor)
		r2.SetReplyTo(msg)
		return r2, nil
	}

	r.Description = descStr
	r.SetColor(okColor)

	return r, nil
}

func (c *UserCommands) withdraw(ctx context.Context, logger log.Logger, msg msghandler.MessageLike, gsettings storage.GuildSettings, checkChannel bool, gid, uid snowflake.Snowflake, eventName string) (r2 *cmdhandler.EmbedResponse, err error) {
	t, err := c.deps.TrialAPI().NewTransaction(ctx, msg.GuildID().ToString(), true)
	if err != nil {
		return nil, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	trial, err := t.GetTrial(ctx, eventName)
	if err != nil {
		return nil, err
	}

	signupCidStr := trial.GetSignupChannel(ctx)

	if checkChannel {
		if !isSignupChannel(ctx, logger, msg, signupCidStr, gsettings.AdminChannel, gsettings.AdminRoles, c.deps.BotSession(), c.deps.Bot()) {
			level.Info(logger).Message("command not in signup channel", "signup_channel", trial.GetSignupChannel(ctx))
			return nil, msghandler.ErrNoResponse
		}
	}

	if trial.GetState(ctx) != storage.TrialStateOpen {
		return nil, errors.New("cannot withdraw from a closed trial")
	}

	trial.RemoveSignup(ctx, cmdhandler.UserMentionString(msg.UserID()))

	if err = t.SaveTrial(ctx, trial); err != nil {
		return nil, errors.Wrap(err, "could not save trial withdraw")
	}

	if err = t.Commit(ctx); err != nil {
		return nil, errors.Wrap(err, "could not save trial withdraw")
	}

	if gsettings.ShowAfterWithdraw == "true" {
		level.Debug(logger).Message("auto-show after withdraw", "trial_name", eventName)

		var signupCid snowflake.Snowflake

		sessionGuild, ok := c.deps.BotSession().Guild(gid)
		if ok {
			if scID, ok := sessionGuild.ChannelWithName(signupCidStr); ok {
				signupCid = scID
			}

			r2 = formatTrialDisplay(ctx, trial, true)
			r2.ToChannel = signupCid
		}
	}

	return r2, nil
}
