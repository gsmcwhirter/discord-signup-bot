package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gsmcwhirter/go-util/v3/deferutil"
	"github.com/gsmcwhirter/go-util/v3/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v7/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v7/logging"
)

func (c *userCommands) list(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.EmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling rootCommand", "command", "list")

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	t, err := c.deps.TrialAPI().NewTransaction(msg.GuildID().ToString(), false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	g, ok := c.deps.BotSession().Guild(msg.GuildID())
	if !ok {
		return r, ErrGuildNotFound
	}

	trials := t.GetTrials()
	tNames := make([]string, 0, len(trials))
	for _, trial := range trials {
		if trial.GetState() != storage.TrialStateClosed {
			if tscID, ok := g.ChannelWithName(trial.GetSignupChannel()); ok {
				tNames = append(tNames, fmt.Sprintf("%s (%s)", trial.GetName(), cmdhandler.ChannelMentionString(tscID)))
			} else {
				tNames = append(tNames, trial.GetName())
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
