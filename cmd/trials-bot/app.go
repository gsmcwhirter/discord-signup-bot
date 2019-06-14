package main

import (
	"context"

	_ "net/http/pprof"

	"github.com/go-kit/kit/log/level"
	"github.com/gsmcwhirter/go-util/v2/deferutil"
	"github.com/gsmcwhirter/go-util/v2/pprofsidecar"

	"github.com/gsmcwhirter/discord-bot-lib/v6/bot"
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

	botConfig := bot.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		BotToken:     c.ClientToken,
		APIURL:       c.DiscordAPI,
		NumWorkers:   c.NumWorkers,

		OS:          "linux",
		BotName:     c.BotName,
		BotPresence: c.BotPresence,
	}

	b := bot.NewDiscordBot(deps, botConfig)
	err = b.AuthenticateAndConnect()
	if err != nil {
		return err
	}
	defer deferutil.CheckDefer(b.Disconnect)

	deps.MessageHandler().ConnectToBot(b)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = pprofsidecar.Run(ctx, c.PProfHostPort, nil, b.Run)

	_ = level.Error(deps.Logger()).Log("message", "error in start; quitting", "err", err)
	return err
}
