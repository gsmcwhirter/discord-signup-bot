package commands

import (
	"fmt"

	"github.com/gsmcwhirter/go-util/v7/deferutil"
	"github.com/gsmcwhirter/go-util/v7/errors"
	"github.com/gsmcwhirter/go-util/v7/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/reactions"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v18/cmdhandler"
)

func (c *reactionHandler) withdraw(msg reactions.Reaction) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "userCommands.withdraw", "guild_id", msg.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := reactions.LoggerWithReaction(msg, c.deps.Logger())
	level.Info(logger).Message("handling reaction", "command", "withdraw")

	trialName, err := c.getTrialNameForReaction(ctx, logger, msg)
	if err != nil {
		return r, msghandler.ErrNoResponse
	}
	level.Info(logger).Message("reaction event identified", "trial_name", trialName)

	// proceed with the withdraw
	gsettings, err := storage.GetSettings(ctx, c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	t, err := c.deps.TrialAPI().NewTransaction(ctx, msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	trial, err := t.GetTrial(ctx, trialName)
	if err != nil {
		return r, err
	}

	if !isSignupChannel(ctx, logger, msg, trial.GetSignupChannel(ctx), gsettings.AdminChannel, gsettings.AdminRole, c.deps.BotSession(), c.deps.Bot()) {
		level.Info(logger).Message("command not in signup channel", "signup_channel", trial.GetSignupChannel(ctx))
		return r, msghandler.ErrNoResponse
	}

	if trial.GetState(ctx) != storage.TrialStateOpen {
		return r, errors.New("cannot withdraw from a closed trial")
	}

	trial.RemoveSignup(ctx, cmdhandler.UserMentionString(msg.UserID()))

	if err = t.SaveTrial(ctx, trial); err != nil {
		return r, errors.Wrap(err, "could not save trial withdraw")
	}

	if err = t.Commit(ctx); err != nil {
		return r, errors.Wrap(err, "could not save trial withdraw")
	}

	level.Info(logger).Message("withdrew", "trial_name", trialName)
	descStr := fmt.Sprintf("Withdrew from %s", trialName)

	if gsettings.ShowAfterWithdraw == "true" {
		level.Debug(logger).Message("auto-show after withdraw", "trial_name", trialName)

		r2 := formatTrialDisplay(ctx, trial, true)
		r2.To = cmdhandler.UserMentionString(msg.UserID())
		r2.Description = fmt.Sprintf("%s\n\n%s", descStr, r2.Description)
		return r2, nil
	}

	r.Description = descStr

	return r, nil
}
