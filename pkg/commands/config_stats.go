package commands

import (
	"context"
	"fmt"

	"github.com/gsmcwhirter/go-util/v5/deferutil"
	"github.com/gsmcwhirter/go-util/v5/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v10/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v10/logging"
)

func (c *configCommands) collectStats(ctx context.Context, gid string) (stat, error) {
	s := stat{}

	t, err := c.deps.TrialAPI().NewTransaction(ctx, gid, false)
	if err != nil {
		return s, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	trials := t.GetTrials(ctx)

	for _, trial := range trials {
		s.trials++
		if trial.GetState(ctx) == storage.TrialStateClosed {
			s.closed++
		} else {
			s.open++
		}
	}

	return s, nil
}

func (c *configCommands) stats(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "configCommands.stats", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling configCommand", "command", "stats")

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	allGuilds, err := c.deps.GuildAPI().AllGuilds(msg.Context())
	if err != nil {
		return r, err
	}

	s := stat{}

	for _, guild := range allGuilds {
		st, err := c.collectStats(msg.Context(), guild)
		if err != nil {
			return r, err
		}

		s.trials += st.trials
		s.open += st.open
		s.closed += st.closed
	}

	r.Description = fmt.Sprintf("Total guilds: %d\nTotal events: %d\nCurrently open: %d\nCurrently closed: %d\n", len(allGuilds), s.trials, s.open, s.closed)
	return r, nil
}
