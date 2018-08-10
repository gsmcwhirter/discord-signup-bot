package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	bolt "github.com/coreos/bbolt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gsmcwhirter/discord-bot-lib/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/discordapi"
	"github.com/gsmcwhirter/discord-bot-lib/discordapi/messagehandler"
	"github.com/gsmcwhirter/discord-bot-lib/discordapi/session"
	"github.com/gsmcwhirter/discord-bot-lib/httpclient"
	"github.com/gsmcwhirter/discord-bot-lib/wsclient"
	"golang.org/x/time/rate"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/commands"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

type dependencies struct {
	logger             log.Logger
	db                 *bolt.DB
	trialAPI           storage.TrialAPI
	guildAPI           storage.GuildAPI
	httpClient         httpclient.HTTPClient
	wsClient           wsclient.WSClient
	cmdHandler         *cmdhandler.CommandHandler
	configHandler      *cmdhandler.CommandHandler
	adminHandler       *cmdhandler.CommandHandler
	discordMsgHandler  discordapi.DiscordMessageHandler
	messageRateLimiter *rate.Limiter
	connectRateLimiter *rate.Limiter
	msgHandlers        msghandler.Handlers
	botSession         *session.Session
}

func createDependencies(conf config) (d *dependencies, err error) {
	d = &dependencies{}

	var logger log.Logger
	if conf.LogFormat == "json" {
		logger = log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	} else {
		logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	}

	switch conf.LogLevel {
	case "debug":
		logger = level.NewFilter(logger, level.AllowDebug())
	case "info":
		logger = level.NewFilter(logger, level.AllowInfo())
	case "warn":
		logger = level.NewFilter(logger, level.AllowWarn())
	case "error":
		logger = level.NewFilter(logger, level.AllowError())
	default:
		logger = level.NewFilter(logger, level.AllowAll())
	}

	logger = log.With(logger, "timestamp", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	d.logger = logger

	d.db, err = bolt.Open(conf.Database, 0660, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return
	}

	d.trialAPI, err = storage.NewBoltTrialAPI(d.db)
	if err != nil {
		return
	}

	d.guildAPI, err = storage.NewBoltGuildAPI(d.db)
	if err != nil {
		return
	}

	d.httpClient = httpclient.NewHTTPClient(d)
	h := http.Header{}
	h.Add("User-Agent", fmt.Sprintf("DiscordBot (%s, %s)", conf.ClientURL, BuildVersion))
	h.Add("Authorization", fmt.Sprintf("Bot %s", conf.ClientToken))
	d.httpClient.SetHeaders(h)

	d.wsClient = wsclient.NewWSClient(d, wsclient.Options{MaxConcurrentHandlers: conf.NumWorkers})

	d.cmdHandler = commands.CommandHandler(d, conf.Version, commands.Options{CmdIndicator: " "})
	d.configHandler = commands.ConfigHandler(d, conf.Version, commands.Options{CmdIndicator: " "})
	d.adminHandler = commands.AdminHandler(d, conf.Version, commands.Options{CmdIndicator: " "})

	d.connectRateLimiter = rate.NewLimiter(rate.Every(5*time.Second), 1)
	d.messageRateLimiter = rate.NewLimiter(rate.Every(60*time.Second), 120)

	d.discordMsgHandler = messagehandler.NewDiscordMessageHandler(d)
	d.botSession = session.NewSession()

	d.msgHandlers = msghandler.NewHandlers(d, msghandler.Options{
		DefaultCommandIndicator: "!",
		ErrorColor:              0xff0000,
		SuccessColor:            0xaa63ff,
	})

	return
}

func (d *dependencies) Close() {
	if d.db != nil {
		d.db.Close() // nolint: errcheck
	}

	if d.wsClient != nil {
		d.wsClient.Close()
	}
}

func (d *dependencies) Logger() log.Logger {
	return d.logger
}

func (d *dependencies) HTTPClient() httpclient.HTTPClient {
	return d.httpClient
}

func (d *dependencies) CommandHandler() *cmdhandler.CommandHandler {
	return d.cmdHandler
}

func (d *dependencies) ConfigHandler() *cmdhandler.CommandHandler {
	return d.configHandler
}

func (d *dependencies) AdminHandler() *cmdhandler.CommandHandler {
	return d.adminHandler
}

func (d *dependencies) MessageHandler() msghandler.Handlers {
	return d.msgHandlers
}

func (d *dependencies) WSClient() wsclient.WSClient {
	return d.wsClient
}

func (d *dependencies) GuildAPI() storage.GuildAPI {
	return d.guildAPI
}

func (d *dependencies) TrialAPI() storage.TrialAPI {
	return d.trialAPI
}

func (d *dependencies) MessageRateLimiter() *rate.Limiter {
	return d.messageRateLimiter
}

func (d *dependencies) ConnectRateLimiter() *rate.Limiter {
	return d.connectRateLimiter
}

func (d *dependencies) BotSession() *session.Session {
	return d.botSession
}

func (d *dependencies) DiscordMessageHandler() discordapi.DiscordMessageHandler {
	return d.discordMsgHandler
}
