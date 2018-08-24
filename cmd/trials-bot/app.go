package main

import (
	"context"

	_ "net/http/pprof"

	"github.com/go-kit/kit/log/level"

	"github.com/gsmcwhirter/discord-bot-lib/discordapi"
	"github.com/gsmcwhirter/go-util/pprofsidecar"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = pprofsidecar.Run(ctx, c.PProfHostPort, nil, bot.Run)

	_ = level.Error(deps.Logger()).Log("message", "error in start; quitting", "err", err)
	return err
}
