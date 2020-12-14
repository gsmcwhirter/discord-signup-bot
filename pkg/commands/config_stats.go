package commands

import (
	"context"
	"fmt"
	"runtime"

	"github.com/dustin/go-humanize"
	"github.com/gsmcwhirter/discord-bot-lib/v18/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v18/logging"
	"github.com/gsmcwhirter/go-util/v7/deferutil"
	"github.com/gsmcwhirter/go-util/v7/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
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
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	if msg.UserID().ToString() != "183367875350888466" {
		return r, msghandler.ErrUnauthorized
	}

	r.SetReplyTo(msg)

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

	gids := c.deps.BotSession().GuildIDs()

	ms := runtime.MemStats{}
	runtime.ReadMemStats(&ms)

	bc := c.deps.Bot().Config()

	r.Description = fmt.Sprintf(`
Database stats:
%[1]s
Total guilds: %[2]d
Total events: %[3]d
Currently open: %[4]d
Currently closed: %[5]d
%[1]s

Session stats:
%[1]s
Total guilds: %[6]d
%[1]s

Runtime stats:
%[1]s
GOMAXPROCS: %[7]d
CPUs: %[8]d
Goroutines: %[9]d
HeapAlloc: %[10]s
Active Objects: %[11]d
Last GC Pause Duration: %[12]d ns
%[1]s

Bot Configuration:
%[1]s
Client ID: %[13]s
Num Workers: %[14]d
%[1]s

Recent Stats:
%[1]s
%[15]s
%[1]s

`, "```",
		// db stats
		len(allGuilds),
		s.trials,
		s.open,
		s.closed,
		// session stats
		len(gids),
		// runtime stats
		runtime.GOMAXPROCS(0),
		runtime.NumCPU(),
		runtime.NumGoroutine(),
		humanize.Bytes(ms.HeapAlloc),
		ms.Mallocs-ms.Frees,
		ms.PauseNs[(ms.NumGC+255)%256],
		bc.ClientID,
		bc.NumWorkers,
		// rolling stats
		c.deps.StatsHub().Report(""),
	)
	return r, nil
}
