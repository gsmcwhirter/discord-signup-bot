package main

import (
	"context"
	"net/http"
	"time"

	// _ "net/http/pprof"

	"github.com/gsmcwhirter/go-util/v7/deferutil"
	"github.com/gsmcwhirter/go-util/v7/logging/level"
	"github.com/gsmcwhirter/go-util/v7/pprofsidecar"
	"golang.org/x/sync/errgroup"

	"github.com/gsmcwhirter/discord-bot-lib/v18/bot"
)

type config struct {
	BotID               string  `mapstructure:"bot_id"`
	BotName             string  `mapstructure:"bot_name"`
	BotPresence         string  `mapstructure:"bot_presence"`
	DiscordAPI          string  `mapstructure:"discord_api"`
	ClientID            string  `mapstructure:"client_id"`
	ClientSecret        string  `mapstructure:"client_secret"`
	ClientToken         string  `mapstructure:"client_token"`
	Database            string  `mapstructure:"database"`
	ClientURL           string  `mapstructure:"client_url"`
	LogFormat           string  `mapstructure:"log_format"`
	LogLevel            string  `mapstructure:"log_level"`
	PProfHostPort       string  `mapstructure:"pprof_hostport"`
	Version             string  `mapstructure:"-"`
	NumWorkers          int     `mapstructure:"num_workers"`
	BugsnagAPIKey       string  `mapstructure:"bugsnag_api_key"`
	BugsnagReleaseStage string  `mapstructure:"bugsnag_release_stage"`
	HoneycombAPIKey     string  `mapstructure:"honeycomb_api_key"`
	HoneycombDataset    string  `mapstructure:"honeycomb_dataset"`
	TraceProbability    float64 `mapstructure:"trace_probability"`
	PrometheusNamespace string  `mapstructure:"prometheus_namespace"`
	PrometheusHostPort  string  `mapstructure:"prometheus_hostport"`
}

func start(c config) error {
	// See https://discordapp.com/developers/docs/topics/permissions#permissions-bitwise-permission-flags
	botPermissions := 0x00000040 // add reactions
	botPermissions |= 0x00000400 // view channel (including read messages)
	botPermissions |= 0x00000800 // send messages
	botPermissions |= 0x00002000 // manage messages
	botPermissions |= 0x00004000 // embed links
	botPermissions |= 0x00008000 // attach files
	botPermissions |= 0x00010000 // read message history
	botPermissions |= 0x00020000 // mention everyone
	botPermissions |= 0x04000000 // change own nickname

	botIntents := 1 << 0  // guilds
	botIntents |= 1 << 1  // guild members
	botIntents |= 1 << 9  // guild messages
	botIntents |= 1 << 10 // guild message reactions
	botIntents |= 1 << 12 // direct messages
	botIntents |= 1 << 13 // direct message reactions

	deps, err := createDependencies(c, botPermissions, botIntents)
	if err != nil {
		return err
	}
	defer deps.Close()

	err = deps.Bot().AuthenticateAndConnect()
	if err != nil {
		return err
	}
	defer deferutil.CheckDefer(deps.Bot().Disconnect)

	deps.MessageHandler().ConnectToBot(deps.Bot())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := http.NewServeMux()
	if deps.promHandler != nil {
		mux.Handle("/metrics", deps.promHandler)
	}

	prom := &http.Server{
		Addr:         c.PrometheusHostPort,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		Handler:      mux,
	}

	err = pprofsidecar.Run(ctx, c.PProfHostPort, nil, runAll(deps, deps.Bot(), prom))

	level.Error(deps.Logger()).Err("error in start; quitting", err)
	return err
}

func runAll(deps *dependencies, b bot.DiscordBot, srv *http.Server) func(context.Context) error {
	return func(ctx context.Context) error {
		g, ctx := errgroup.WithContext(ctx)

		g.Go(func() error { return b.Run(ctx) })
		g.Go(serverStartFunc(deps, srv))
		g.Go(serverShutdownFunc(ctx, deps, srv))

		return g.Wait()
	}
}

func serverStartFunc(deps *dependencies, s *http.Server) func() error {
	return func() error {
		level.Info(deps.logger).Message("starting server", "listen", s.Addr)
		return s.ListenAndServe()
	}
}

func serverShutdownFunc(ctx context.Context, deps *dependencies, s *http.Server) func() error {
	return func() error {
		<-ctx.Done() // something said we are done

		level.Info(deps.Logger()).Message("stopping server", "listen", s.Addr)

		shutdownCtx, cncl := context.WithTimeout(context.Background(), 2*time.Second)
		defer cncl()

		return s.Shutdown(shutdownCtx)
	}
}
