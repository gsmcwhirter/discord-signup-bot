package commands

import (
	"fmt"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/logging"
)

func (c *UserCommands) signupHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "userCommands.signup", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling rootCommand", "command", "signup", "trial_and_role", msg.Contents())

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

	if len(msg.Contents()) < 2 {
		return r, errors.New("missing role")
	}

	if len(msg.Contents()) > 2 && len(msg.Contents())%2 != 0 {
		return r, errors.New("incorrect number of arguments")
	}

	t, err := c.deps.TrialAPI().NewTransaction(ctx, msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	var descStr string
	var trial storage.Trial

	for i := 0; i < len(msg.Contents()); i += 2 {
		trialName, role := msg.Contents()[i], msg.Contents()[i+1]

		trial, err = t.GetTrial(ctx, trialName)
		if err != nil {
			return r, err
		}

		if !isSignupChannel(ctx, logger, msg, trial.GetSignupChannel(ctx), gsettings.AdminChannel, gsettings.AdminRoles, c.deps.BotSession(), c.deps.Bot()) {
			level.Info(logger).Message("command not in signup channel", "signup_channel", trial.GetSignupChannel(ctx))
			return r, msghandler.ErrNoResponse
		}

		if trial.GetState(ctx) != storage.TrialStateOpen {
			return r, errors.New("cannot sign up for a closed trial")
		}

		overflow, err := signupUser(ctx, trial, cmdhandler.UserMentionString(msg.UserID()), role)
		if err != nil {
			return r, err
		}

		if err = t.SaveTrial(ctx, trial); err != nil {
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

	if err = t.Commit(ctx); err != nil {
		return r, errors.Wrap(err, "could not save trial signup")
	}

	if gsettings.ShowAfterSignup == "true" {
		if len(msg.Contents()) > 2 {
			descStr += "\n(only showing last trial details)"
		}

		r2 := formatTrialDisplay(ctx, trial, true)
		// r2.To = cmdhandler.UserMentionString(msg.UserID())
		r2.Description = fmt.Sprintf("%s\n\n%s", descStr, r2.Description)
		r2.SetReplyTo(msg)
		r2.Color = okColor
		return r2, nil
	}

	r.Description = descStr
	r.Color = okColor

	return r, nil
}
