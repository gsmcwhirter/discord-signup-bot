package commands

import (
	"strings"

	"github.com/gsmcwhirter/go-util/v3/deferutil"
	"github.com/gsmcwhirter/go-util/v3/errors"
	"github.com/gsmcwhirter/go-util/v3/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v7/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v7/logging"
)

func (c *userCommands) show(msg cmdhandler.Message) (cmdhandler.Response, error) {
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

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	trial, err := t.GetTrial(trialName)
	if err != nil {
		return r, err
	}

	if !isSignupChannel(logger, msg, trial.GetSignupChannel(), gsettings.AdminChannel, gsettings.AdminRole, c.deps.BotSession()) {
		level.Info(logger).Message("command not in signup channel", "signup_channel", trial.GetSignupChannel())
		return r, msghandler.ErrNoResponse
	}

	r2 := formatTrialDisplay(trial, true)
	r2.To = cmdhandler.UserMentionString(msg.UserID())

	return r2, nil
}
