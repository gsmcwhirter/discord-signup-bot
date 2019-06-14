package commands

import (
	"fmt"

	"github.com/go-kit/kit/log/level"
	"github.com/gsmcwhirter/go-util/v2/deferutil"
	"github.com/pkg/errors"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v6/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v6/logging"
)

func (c *adminCommands) delete(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	_ = level.Info(logger).Log("message", "handling adminCommand", "command", "delete", "args", msg.Contents())

	gsettings, err := storage.GetSettings(c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		_ = level.Info(logger).Log("message", "command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return nil, msghandler.ErrUnauthorized
	}

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 1 {
		return r, errors.New("need event name")
	}

	if len(msg.Contents()) > 1 {
		return r, errors.New("too many arguments")
	}

	trialName := msg.Contents()[0]

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	if err = t.DeleteTrial(trialName); err != nil {
		return r, errors.Wrap(err, "could not delete event")
	}

	if err = t.Commit(); err != nil {
		return r, errors.Wrap(err, "could not delete event")
	}

	_ = level.Info(logger).Log("message", "trial deleted", "trial_name", trialName)
	r.Description = fmt.Sprintf("Deleted event %q", trialName)

	return r, nil
}
