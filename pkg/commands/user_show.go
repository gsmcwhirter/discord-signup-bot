package commands

import (
	"strings"

	"github.com/gsmcwhirter/go-util/v5/deferutil"
	"github.com/gsmcwhirter/go-util/v5/errors"
	"github.com/gsmcwhirter/go-util/v5/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v10/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v10/logging"
)

func (c *userCommands) show(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "userCommands.show", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling rootCommand", "command", "show", "trial_name", msg.Contents())

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) != 1 {
		return r, errors.New("you must supply exactly 1 argument -- trial name; are you missing quotes?")
	}

	trialName := strings.TrimSpace(msg.Contents()[0])

	gsettings, err := storage.GetSettings(msg.Context(), c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.Context(), msg.GuildID().ToString(), false)
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

	r2 := formatTrialDisplay(msg.Context(), trial, true)
	r2.To = cmdhandler.UserMentionString(msg.UserID())

	return r2, nil
}
