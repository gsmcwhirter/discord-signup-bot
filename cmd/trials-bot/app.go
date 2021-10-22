package main

import (
	"context"
	"net/http"
	"time"

	// _ "net/http/pprof"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/logging/level"
	"github.com/gsmcwhirter/go-util/v8/pprofsidecar"
	"golang.org/x/sync/errgroup"

	"github.com/gsmcwhirter/discord-bot-lib/v20/bot"
)

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

	level.Info(deps.Logger()).Message("pprof hostport", "val", c.PProfHostPort)
	err = pprofsidecar.Run(ctx, c.PProfHostPort, nil, runAll(deps, deps.Bot(), prom))

	level.Error(deps.Logger()).Err("error in start; quitting", err)
	return err
}

func runAll(deps *dependencies, b *bot.DiscordBot, srv *http.Server) func(context.Context) error {
	return func(ctx context.Context) error {
		g, ctx := errgroup.WithContext(ctx)

		g.Go(func() error { return b.Run(ctx) })
		g.Go(serverStartFunc(deps, srv))
		g.Go(serverShutdownFunc(ctx, deps, srv))
		g.Go(func() error { return deps.statsHub.Start(ctx) })

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
