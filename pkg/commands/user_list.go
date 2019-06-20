package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gsmcwhirter/go-util/v5/deferutil"
	"github.com/gsmcwhirter/go-util/v5/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v10/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v10/logging"
)

func (c *userCommands) list(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "userCommands.list", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.EmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling rootCommand", "command", "list")

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.Context(), msg.GuildID().ToString(), false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(msg.Context()) })

	g, ok := c.deps.BotSession().Guild(msg.GuildID())
	if !ok {
		return r, ErrGuildNotFound
	}

	trials := t.GetTrials(msg.Context())
	tNames := make([]string, 0, len(trials))
	for _, trial := range trials {
		if trial.GetState(msg.Context()) != storage.TrialStateClosed {
			if tscID, ok := g.ChannelWithName(trial.GetSignupChannel(msg.Context())); ok {
				tNames = append(tNames, fmt.Sprintf("%s (%s)", trial.GetName(msg.Context()), cmdhandler.ChannelMentionString(tscID)))
			} else {
				tNames = append(tNames, trial.GetName(msg.Context()))
			}
		}
	}
	sort.Strings(tNames)

	var listContent string
	if len(tNames) > 0 {
		listContent = strings.Join(tNames, "\n")
	} else {
		listContent = "(none yet)"
	}

	r.Fields = []cmdhandler.EmbedField{
		{
			Name: "*Available Trials*",
			Val:  listContent,
		},
	}

	return r, nil
}
