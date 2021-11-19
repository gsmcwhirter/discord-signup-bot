package commands

import (
	"context"
	"fmt"
	"runtime"

	"github.com/dustin/go-humanize"
	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
	"github.com/gsmcwhirter/discord-bot-lib/v23/logging"
	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

func (c *ConfigCommands) collectStats(ctx context.Context, gid string) (stat, error) {
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

func (c *ConfigCommands) statsInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "configCommands.statsInteraction")
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	if ix.UserID().ToString() != "183367875350888466" {
		return r, nil, msghandler.ErrUnauthorized
	}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling config interaction", "command", "stats")

	gsettings, err := storage.GetSettings(ctx, c.deps.GuildAPI(), ix.GuildID())
	if err != nil {
		return r, nil, err
	}

	okColor, err := colorToInt(gsettings.MessageColor)
	if err != nil {
		return r, nil, err
	}

	errColor, err := colorToInt(gsettings.ErrorColor)
	if err != nil {
		return r, nil, err
	}

	r.SetColor(errColor)

	if !isAdminChannel(logger, ix, gsettings.AdminChannel, c.deps.BotSession()) {
		level.Info(logger).Message("command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, nil, msghandler.ErrUnauthorized
	}

	level.Info(logger).Message("sending deferral")

	if err := c.deps.Bot().API().DeferInteractionResponse(ctx, ix.IDSnowflake, ix.Token); err != nil {
		return r, nil, errors.Wrap(err, "failed to send deferral request")
	}

	level.Debug(logger).Message("collecting statistics")

	r2, err := c.stats(ctx)
	if err != nil {
		return r, nil, errors.Wrap(err, "could not collect stats")
	}

	level.Debug(logger).Message("done collecting statistics")

	r2.SetColor(okColor)

	return r2, nil, nil
}

func (c *ConfigCommands) statsHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "configCommands.statsHandler")
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

	gsettings, err := storage.GetSettings(ctx, c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	okColor, err := colorToInt(gsettings.MessageColor)
	if err != nil {
		return r, err
	}

	errColor, err := colorToInt(gsettings.ErrorColor)
	if err != nil {
		return r, err
	}

	r.SetColor(errColor)

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		level.Info(logger).Message("command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, msghandler.ErrUnauthorized
	}

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	r2, err := c.stats(ctx)
	if err != nil {
		return r, errors.Wrap(err, "could not collect stats")
	}

	r2.SetColor(okColor)

	return r2, nil
}

func (c *ConfigCommands) stats(ctx context.Context) (*cmdhandler.SimpleEmbedResponse, error) {
	ctx, span := c.deps.Census().StartSpan(ctx, "configCommands.stats")
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	allGuilds, err := c.deps.GuildAPI().AllGuilds(ctx)
	if err != nil {
		return r, err
	}

	s := stat{}

	for _, guild := range allGuilds {
		st, err := c.collectStats(ctx, guild)
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
