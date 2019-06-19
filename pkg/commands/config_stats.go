package commands

import (
	"context"
	"fmt"

	"github.com/gsmcwhirter/go-util/v4/deferutil"
	"github.com/gsmcwhirter/go-util/v4/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v9/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v9/logging"
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
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "configCommands.stats")
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
		stat, err := c.collectStats(msg.Context(), guild)
		if err != nil {
			return r, err
		}

		s.trials += stat.trials
		s.open += stat.open
		s.closed += stat.closed
	}

	r.Description = fmt.Sprintf("Total guilds: %d\nTotal events: %d\nCurrently open: %d\nCurrently closed: %d\n", len(allGuilds), s.trials, s.open, s.closed)
	return r, nil
}
