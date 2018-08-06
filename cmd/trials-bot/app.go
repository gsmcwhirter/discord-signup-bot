package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-kit/kit/log/level"
	"golang.org/x/sync/errgroup"

	"github.com/gsmcwhirter/discord-bot-lib/discordapi"
)

type config struct {
	BotName       string `mapstructure:"bot_name"`
	BotPresence   string `mapstructure:"bot_presence"`
	DiscordAPI    string `mapstructure:"discord_api"`
	ClientID      string `mapstructure:"client_id"`
	ClientSecret  string `mapstructure:"client_secret"`
	ClientToken   string `mapstructure:"client_token"`
	Database      string `mapstructure:"database"`
	ClientURL     string `mapstructure:"client_url"`
	LogFormat     string `mapstructure:"log_format"`
	LogLevel      string `mapstructure:"log_level"`
	PProfHostPort string `mapstructure:"pprof_hostport"`
	Version       string `mapstructure:"-"`
	NumWorkers    int    `mapstructure:"num_workers"`
}

func start(c config) error {
	deps, err := createDependencies(c)
	if err != nil {
		return err
	}
	defer deps.Close()

	botConfig := discordapi.BotConfig{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		BotToken:     c.ClientToken,
		APIURL:       c.DiscordAPI,
		NumWorkers:   c.NumWorkers,

		OS:          "linux",
		BotName:     c.BotName,
		BotPresence: c.BotPresence,
	}

	bot := discordapi.NewDiscordBot(deps, botConfig)
	err = bot.AuthenticateAndConnect()
	if err != nil {
		return err
	}
	defer bot.Disconnect() // nolint: errcheck

	deps.MessageHandler().ConnectToBot(bot)

	interrupt := make(chan os.Signal)
	defer close(interrupt)
	signal.Notify(interrupt, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)
	srv := &http.Server{Addr: c.PProfHostPort} // the pprof debug server

	// watches for interrupts
	g.Go(func() error {
		select {
		case <-interrupt:
			cancel()
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	// runs the bot
	g.Go(func() error {
		return bot.Run(ctx)
	})

	// runs the pprof server
	g.Go(srv.ListenAndServe)

	// kills the pprof server when necessary
	g.Go(func() error {
		<-ctx.Done() // something said we are done

		shutdownCtx, cncl := context.WithTimeout(context.Background(), 2*time.Second)
		defer cncl()

		return srv.Shutdown(shutdownCtx)
	})

	err = g.Wait()
	_ = level.Error(deps.Logger()).Log("message", "error in start; quitting", "err", err)
	return err
}
