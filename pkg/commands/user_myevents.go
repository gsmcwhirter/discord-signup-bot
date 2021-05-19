package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gsmcwhirter/go-util/v7/deferutil"
	"github.com/gsmcwhirter/go-util/v7/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v18/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v18/logging"
)

func (c *userCommands) myEvents(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "userCommands.myEvents", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.EmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling rootCommand", "command", "myEvents")

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
		if trial.GetState(msg.Context()) == storage.TrialStateClosed {
			continue
		}

		signups := trial.GetSignups(msg.Context())
		signedUp := false
		role := ""
		for _, su := range signups {
			if su.GetName(msg.Context()) == cmdhandler.UserMentionString(msg.UserID()) {
				signedUp = true
				role = su.GetRole(msg.Context())
				break
			}
		}

		if !signedUp {
			continue
		}

		if tscID, ok := g.ChannelWithName(trial.GetSignupChannel(msg.Context())); ok {
			tNames = append(tNames, fmt.Sprintf("%s as %s (%s)", trial.GetName(msg.Context()), role, cmdhandler.ChannelMentionString(tscID)))
		} else {
			tNames = append(tNames, fmt.Sprintf("%s as %s", trial.GetName(msg.Context()), role))
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
			Name: "*Registered Events*",
			Val:  listContent,
		},
	}

	return r, nil
}
