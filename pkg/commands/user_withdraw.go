package commands

import (
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v4/deferutil"
	"github.com/gsmcwhirter/go-util/v4/errors"
	"github.com/gsmcwhirter/go-util/v4/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v9/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v9/logging"
)

func (c *userCommands) withdraw(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "userCommands.withdraw", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling rootCommand", "command", "withdraw", "trial_name", msg.Contents())

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 1 {
		return r, errors.New("missing event name")
	}

	trialName := strings.TrimSpace(msg.Contents()[0])

	gsettings, err := storage.GetSettings(msg.Context(), c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.Context(), msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(msg.Context()) })

	trial, err := t.GetTrial(msg.Context(), trialName)
	if err != nil {
		return r, err
	}

	if !isSignupChannel(logger, msg, trial.GetSignupChannel(msg.Context()), gsettings.AdminChannel, gsettings.AdminRole, c.deps.BotSession()) {
		level.Info(logger).Message("command not in signup channel", "signup_channel", trial.GetSignupChannel(msg.Context()))
		return r, msghandler.ErrNoResponse
	}

	if trial.GetState(msg.Context()) != storage.TrialStateOpen {
		return r, errors.New("cannot withdraw from a closed trial")
	}

	trial.RemoveSignup(msg.Context(), cmdhandler.UserMentionString(msg.UserID()))

	if err = t.SaveTrial(msg.Context(), trial); err != nil {
		return r, errors.Wrap(err, "could not save trial withdraw")
	}

	if err = t.Commit(msg.Context()); err != nil {
		return r, errors.Wrap(err, "could not save trial withdraw")
	}

	level.Info(logger).Message("withdrew", "trial_name", trialName)
	descStr := fmt.Sprintf("Withdrew from %s", trialName)

	if gsettings.ShowAfterWithdraw == "true" {
		level.Debug(logger).Message("auto-show after withdraw", "trial_name", trialName)

		r2 := formatTrialDisplay(msg.Context(), trial, true)
		r2.To = cmdhandler.UserMentionString(msg.UserID())
		r2.Description = fmt.Sprintf("%s\n\n%s", descStr, r2.Description)
		return r2, nil
	}

	r.Description = descStr

	return r, nil
}
