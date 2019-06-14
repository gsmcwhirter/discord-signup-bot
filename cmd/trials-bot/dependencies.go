package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	bolt "github.com/coreos/bbolt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"

	"github.com/gsmcwhirter/discord-bot-lib/v6/bot"
	"github.com/gsmcwhirter/discord-bot-lib/v6/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v6/etfapi"
	"github.com/gsmcwhirter/discord-bot-lib/v6/httpclient"
	"github.com/gsmcwhirter/discord-bot-lib/v6/messagehandler"
	"github.com/gsmcwhirter/discord-bot-lib/v6/wsclient"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/commands"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

type dependencies struct {
	logger log.Logger

	db       *bolt.DB
	trialAPI storage.TrialAPI
	guildAPI storage.GuildAPI

	httpDoer   httpclient.Doer
	httpClient httpclient.HTTPClient
	wsDialer   wsclient.Dialer
	wsClient   wsclient.WSClient

	messageRateLimiter *rate.Limiter
	connectRateLimiter *rate.Limiter
	botSession         *etfapi.Session

	cmdHandler        *cmdhandler.CommandHandler
	configHandler     *cmdhandler.CommandHandler
	adminHandler      *cmdhandler.CommandHandler
	discordMsgHandler bot.DiscordMessageHandler
	msgHandlers       msghandler.Handlers
}

func createDependencies(conf config) (*dependencies, error) {
	var err error

	d := &dependencies{
		httpDoer:           &http.Client{},
		wsDialer:           wsclient.WrapDialer(websocket.DefaultDialer),
		connectRateLimiter: rate.NewLimiter(rate.Every(5*time.Second), 1),
		messageRateLimiter: rate.NewLimiter(rate.Every(60*time.Second), 120),
		botSession:         etfapi.NewSession(),
	}

	var logger log.Logger
	if conf.LogFormat == "json" {
		logger = log.NewJSONLogger(log.NewSyncWriter(os.Stderr))
	} else {
		logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
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
		return d, err
	}

	d.trialAPI, err = storage.NewBoltTrialAPI(d.db)
	if err != nil {
		return d, err
	}

	d.guildAPI, err = storage.NewBoltGuildAPI(d.db)
	if err != nil {
		return d, err
	}

	d.httpClient = httpclient.NewHTTPClient(d)
	h := http.Header{}
	h.Add("User-Agent", fmt.Sprintf("DiscordBot (%s, %s)", conf.ClientURL, BuildVersion))
	h.Add("Authorization", fmt.Sprintf("Bot %s", conf.ClientToken))
	d.httpClient.SetHeaders(h)

	d.wsClient = wsclient.NewWSClient(d, wsclient.Options{MaxConcurrentHandlers: conf.NumWorkers})

	d.cmdHandler, err = commands.CommandHandler(d, conf.Version, commands.Options{CmdIndicator: "!"})
	if err != nil {
		return d, err
	}
	d.configHandler, err = commands.ConfigHandler(d, conf.Version, commands.Options{CmdIndicator: "!"})
	if err != nil {
		return d, err
	}
	d.adminHandler, err = commands.AdminHandler(d, conf.Version, commands.Options{CmdIndicator: "!"})
	if err != nil {
		return d, err
	}

	d.discordMsgHandler = messagehandler.NewDiscordMessageHandler(d)

	d.msgHandlers = msghandler.NewHandlers(d, msghandler.Options{
		DefaultCommandIndicator: "!",
		ErrorColor:              0xff0000,
		SuccessColor:            0xaa63ff,
	})

	return d, nil
}

func (d *dependencies) Close() {
	if d.db != nil {
		d.db.Close() // nolint: errcheck
	}

	if d.wsClient != nil {
		d.wsClient.Close()
	}
}

func (d *dependencies) Logger() log.Logger                         { return d.logger }
func (d *dependencies) GuildAPI() storage.GuildAPI                 { return d.guildAPI }
func (d *dependencies) TrialAPI() storage.TrialAPI                 { return d.trialAPI }
func (d *dependencies) HTTPDoer() httpclient.Doer                  { return d.httpDoer }
func (d *dependencies) HTTPClient() httpclient.HTTPClient          { return d.httpClient }
func (d *dependencies) WSDialer() wsclient.Dialer                  { return d.wsDialer }
func (d *dependencies) WSClient() wsclient.WSClient                { return d.wsClient }
func (d *dependencies) MessageRateLimiter() *rate.Limiter          { return d.messageRateLimiter }
func (d *dependencies) ConnectRateLimiter() *rate.Limiter          { return d.connectRateLimiter }
func (d *dependencies) BotSession() *etfapi.Session                { return d.botSession }
func (d *dependencies) CommandHandler() *cmdhandler.CommandHandler { return d.cmdHandler }
func (d *dependencies) ConfigHandler() *cmdhandler.CommandHandler  { return d.configHandler }
func (d *dependencies) AdminHandler() *cmdhandler.CommandHandler   { return d.adminHandler }
func (d *dependencies) MessageHandler() msghandler.Handlers        { return d.msgHandlers }
func (d *dependencies) DiscordMessageHandler() bot.DiscordMessageHandler {
	return d.discordMsgHandler
}
