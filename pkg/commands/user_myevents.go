package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v20/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v20/logging"
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

	t, err := c.deps.TrialAPI().NewTransaction(ctx, msg.GuildID().ToString(), false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	g, ok := c.deps.BotSession().Guild(msg.GuildID())
	if !ok {
		return r, ErrGuildNotFound
	}

	trials := t.GetTrials(ctx)
	tNames := make([]string, 0, len(trials))
	for _, trial := range trials {
		if trial.GetState(ctx) == storage.TrialStateClosed {
			continue
		}

		signups := trial.GetSignups(ctx)
		signedUp := false
		role := ""
		for _, su := range signups {
			if su.GetName(ctx) == cmdhandler.UserMentionString(msg.UserID()) {
				signedUp = true
				role = su.GetRole(ctx)
				break
			}
		}

		if !signedUp {
			continue
		}

		if tscID, ok := g.ChannelWithName(trial.GetSignupChannel(ctx)); ok {
			tNames = append(tNames, fmt.Sprintf("%s as %s (%s)", trial.GetName(ctx), role, cmdhandler.ChannelMentionString(tscID)))
		} else {
			tNames = append(tNames, fmt.Sprintf("%s as %s", trial.GetName(ctx), role))
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
