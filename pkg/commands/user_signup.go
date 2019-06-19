package commands

import (
	"fmt"

	"github.com/gsmcwhirter/go-util/v4/deferutil"
	"github.com/gsmcwhirter/go-util/v4/errors"
	"github.com/gsmcwhirter/go-util/v4/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v9/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v9/logging"
)

func (c *userCommands) signup(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "userCommands.signup")
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling rootCommand", "command", "signup", "trial_and_role", msg.Contents())

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 2 {
		return r, errors.New("missing role")
	}

	if len(msg.Contents()) > 2 && len(msg.Contents())%2 != 0 {
		return r, errors.New("incorrect number of arguments")
	}

	gsettings, err := storage.GetSettings(msg.Context(), c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.Context(), msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(msg.Context()) })

	var descStr string
	var trial storage.Trial

	for i := 0; i < len(msg.Contents()); i += 2 {
		trialName, role := msg.Contents()[i], msg.Contents()[i+1]

		trial, err = t.GetTrial(msg.Context(), trialName)
		if err != nil {
			return r, err
		}

		if !isSignupChannel(logger, msg, trial.GetSignupChannel(msg.Context()), gsettings.AdminChannel, gsettings.AdminRole, c.deps.BotSession()) {
			level.Info(logger).Message("command not in signup channel", "signup_channel", trial.GetSignupChannel(msg.Context()))
			return r, msghandler.ErrNoResponse
		}

		if trial.GetState(msg.Context()) != storage.TrialStateOpen {
			return r, errors.New("cannot sign up for a closed trial")
		}

		overflow, err := signupUser(msg.Context(), trial, cmdhandler.UserMentionString(msg.UserID()), role)
		if err != nil {
			return r, err
		}

		if err = t.SaveTrial(msg.Context(), trial); err != nil {
			return r, errors.Wrap(err, "could not save trial signup")
		}

		if overflow {
			level.Info(logger).Message("signed up", "overflow", true, "role", role, "trial_name", trialName)
			descStr += fmt.Sprintf("Signed up as OVERFLOW for %s in %s\n", role, trialName)
		} else {
			level.Info(logger).Message("signed up", "overflow", false, "role", role, "trial_name", trialName)
			descStr += fmt.Sprintf("Signed up for %s in %s\n", role, trialName)
		}

	}

	if err = t.Commit(msg.Context()); err != nil {
		return r, errors.Wrap(err, "could not save trial signup")
	}

	if gsettings.ShowAfterSignup == "true" {
		if len(msg.Contents()) > 2 {
			descStr += "\n(only showing last trial details)"
		}

		r2 := formatTrialDisplay(msg.Context(), trial, true)
		r2.To = cmdhandler.UserMentionString(msg.UserID())
		r2.Description = fmt.Sprintf("%s\n\n%s", descStr, r2.Description)
		return r2, nil
	}

	r.Description = descStr

	return r, nil
}
