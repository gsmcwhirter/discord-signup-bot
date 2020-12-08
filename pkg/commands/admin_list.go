package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gsmcwhirter/go-util/v7/deferutil"
	"github.com/gsmcwhirter/go-util/v7/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v18/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v18/logging"
)

func (c *adminCommands) list(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "adminCommands.list", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.EmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	fmt.Printf("%#v", r)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling adminCommand", "command", "list")

	gsettings, err := storage.GetSettings(msg.Context(), c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		level.Info(logger).Message("command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return nil, msghandler.ErrUnauthorized
	}

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.Context(), msg.GuildID().ToString(), false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(msg.Context()) })

	trials := t.GetTrials(msg.Context())
	tNamesOpen := make([]string, 0, len(trials))
	tNamesClosed := make([]string, 0, len(trials))
	for _, trial := range trials {
		if trial.GetState(msg.Context()) == storage.TrialStateClosed {
			tNamesClosed = append(tNamesClosed, fmt.Sprintf("%s (#%s)", trial.GetName(msg.Context()), trial.GetSignupChannel(msg.Context())))
		} else {
			tNamesOpen = append(tNamesOpen, fmt.Sprintf("%s (#%s)", trial.GetName(msg.Context()), trial.GetSignupChannel(msg.Context())))
		}
	}
	sort.Strings(tNamesOpen)
	sort.Strings(tNamesClosed)

	r.Fields = []cmdhandler.EmbedField{
		{
			Name: "*Available Trials*",
			Val:  fmt.Sprintf("```\n%s\n```\n", strings.Join(tNamesOpen, "\n")),
		},
		{
			Name: "*Closed Trials*",
			Val:  fmt.Sprintf("```\n%s\n```\n", strings.Join(tNamesClosed, "\n")),
		},
	}

	return r, nil
}
