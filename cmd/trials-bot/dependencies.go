package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/gsmcwhirter/discord-bot-lib/v23/bot"
	"github.com/gsmcwhirter/discord-bot-lib/v23/bot/session"
	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/jsonapi"
	"github.com/gsmcwhirter/discord-bot-lib/v23/dispatcher"
	"github.com/gsmcwhirter/discord-bot-lib/v23/errreport"
	"github.com/gsmcwhirter/discord-bot-lib/v23/httpclient"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
	bstats "github.com/gsmcwhirter/discord-bot-lib/v23/stats"
	"github.com/gsmcwhirter/discord-bot-lib/v23/wsapi"
	"github.com/gsmcwhirter/discord-bot-lib/v23/wsclient"
	"github.com/gsmcwhirter/go-util/v8/errors"
	log "github.com/gsmcwhirter/go-util/v8/logging"
	"github.com/gsmcwhirter/go-util/v8/telemetry"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"golang.org/x/time/rate"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/bugsnag"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/commands"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/permissions"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/pgxutil"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/reactions"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/stats"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

const DiscordAPI = "https://discord.com/api/v8"

type Logger = interface {
	Log(keyvals ...interface{}) error
	Message(string, ...interface{})
	Err(string, error, ...interface{})
	Printf(string, ...interface{})
}

type dependencies struct {
	logger Logger

	db       *pgxpool.Pool
	trialAPI storage.TrialAPI
	guildAPI storage.GuildAPI

	httpDoer   httpclient.Doer
	httpClient *httpclient.HTTPClient
	wsDialer   wsclient.Dialer
	wsClient   *wsclient.WSClient
	jsClient   *jsonapi.DiscordJSONClient

	messageRateLimiter   *rate.Limiter
	connectRateLimiter   *rate.Limiter
	reactionsRateLimiter *rate.Limiter
	botSession           *session.Session

	cmdHandler            *cmdhandler.CommandHandler
	configHandler         *cmdhandler.CommandHandler
	adminHandler          *cmdhandler.CommandHandler
	debugHandler          *cmdhandler.CommandHandler
	reactionHandler       reactions.Handler
	interactionDispatcher *cmdhandler.InteractionDispatcher
	discordMsgHandler     *dispatcher.Dispatcher
	msgHandlers           msghandler.Handlers
	permissionsManager    *permissions.Manager
	guildCmdGen           func(snowflake.Snowflake) ([]cmdhandler.InteractionCommandHandler, error)

	rep         bugsnag.Reporter
	census      *telemetry.Census
	promHandler http.Handler

	bot *bot.DiscordBot

	statsHub *stats.Hub

	sendAllowed            bool
	interactionSendAllowed bool
}

func createDependencies(conf config, botPermissions, botIntents int) (*dependencies, error) {
	var err error

	ctx := context.Background()

	d := &dependencies{
		sendAllowed:            !conf.DisableSends,
		interactionSendAllowed: !conf.DisableInteractionSends,
		httpDoer:               &http.Client{},
		wsDialer:               wsclient.WrapDialer(websocket.DefaultDialer),
		connectRateLimiter:     rate.NewLimiter(rate.Every(5*time.Second), 1),
		messageRateLimiter:     rate.NewLimiter(rate.Every(60*time.Second), 120),
		reactionsRateLimiter:   rate.NewLimiter(rate.Every(500*time.Millisecond), 1),
		botSession:             session.NewSession(),
		statsHub:               stats.NewHub(),
	}

	if err = d.statsHub.Add("raw_msgs", bstats.NewActivityRecorder(30.0)); err != nil {
		return d, errors.Wrap(err, "could not create raw_msgs recorder")
	}
	if err = d.statsHub.Add("msgs", bstats.NewActivityRecorder(30.0)); err != nil {
		return d, errors.Wrap(err, "could not create msgs recorder")
	}
	if err = d.statsHub.Add("reactions", bstats.NewActivityRecorder(30.0)); err != nil {
		return d, errors.Wrap(err, "could not create reactions recorder")
	}
	if err = d.statsHub.Add("msg_sent", bstats.NewActivityRecorder(30.0)); err != nil {
		return d, errors.Wrap(err, "could not create msg_sent recorder")
	}
	if err = d.statsHub.Add("reaction_sent", bstats.NewActivityRecorder(30.0)); err != nil {
		return d, errors.Wrap(err, "could not create reaction_sent recorder")
	}

	var logger Logger
	if conf.LogFormat == "json" {
		logger = log.NewJSONLogger()
	} else {
		logger = log.NewLogfmtLogger()
	}

	logger = log.WithLevel(logger, conf.LogLevel)
	logger = log.With(logger, "timestamp", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	d.logger = logger

	promExp, err := stats.NewPrometheusExporter(stats.PrometheusConfig{
		Namespace: conf.PrometheusNamespace,
	})
	if err != nil {
		return d, err
	}

	d.promHandler = promExp

	cOpts := telemetry.Options{
		StatsExporter: promExp,
		TraceExporter: stats.NewHoneycombExporter(stats.HoneycombConfig{
			APIKey:           conf.HoneycombAPIKey,
			Dataset:          conf.HoneycombDataset,
			TraceProbability: conf.TraceProbability,
		}),
		TraceProbability: conf.TraceProbability,
	}

	d.census = telemetry.NewCensus(cOpts)

	d.rep = bugsnag.NewReporter(logger, conf.BugsnagAPIKey, BuildVersion, conf.BugsnagReleaseStage)

	poolConf, err := pgxpool.ParseConfig(conf.PgDetails)
	if err != nil {
		return d, err
	}

	poolConf.ConnConfig.Logger = &pgxutil.Logger{Logger: d.logger}
	poolConf.ConnConfig.LogLevel = pgx.LogLevelWarn
	poolConf.MaxConnLifetime = 60 * time.Minute
	poolConf.MaxConnIdleTime = 15 * time.Minute
	poolConf.MaxConns = conf.PostgresMaxPoolSize
	poolConf.MinConns = conf.PostgresMinPoolSize
	poolConf.HealthCheckPeriod = 1 * time.Minute

	d.db, err = pgxpool.ConnectConfig(ctx, poolConf)
	if err != nil {
		return d, err
	}

	d.trialAPI, err = storage.NewPgTrialAPI(d.db, d.census)
	if err != nil {
		return d, err
	}

	d.guildAPI, err = storage.NewPgGuildAPI(ctx, d.db, d.census)
	if err != nil {
		return d, err
	}

	d.httpClient = httpclient.NewHTTPClient(d)

	// d.httpClient.SetDebug(true)

	h := http.Header{}
	h.Add("User-Agent", fmt.Sprintf("DiscordBot (%s, %s)", conf.ClientURL, BuildVersion))
	h.Add("Authorization", fmt.Sprintf("Bot %s", conf.ClientToken))
	d.httpClient.SetHeaders(h)

	d.wsClient = wsclient.NewWSClient(d, wsclient.Options{MaxConcurrentHandlers: conf.NumWorkers})
	d.jsClient = jsonapi.NewDiscordJSONClient(d, DiscordAPI)

	d.interactionDispatcher, err = cmdhandler.NewInteractionDispatcher(nil)
	if err != nil {
		return d, err
	}

	d.permissionsManager = permissions.NewManager(d, []string{"config", "admin"})

	var uc *commands.UserCommands
	uc, d.cmdHandler, err = commands.CommandHandler(d, conf.Version, commands.Options{CmdIndicator: "!"})
	if err != nil {
		return d, err
	}

	var cc *commands.ConfigCommands
	cc, d.configHandler, err = commands.ConfigHandler(d, conf.Version, commands.Options{CmdIndicator: "!"})
	if err != nil {
		return d, err
	}
	if err := d.interactionDispatcher.LearnGlobalCommands(cc.GlobalCommands()); err != nil {
		return d, err
	}

	var ac *commands.AdminCommands
	ac, d.adminHandler, err = commands.AdminHandler(d, conf.Version, commands.Options{CmdIndicator: "!"})
	if err != nil {
		return d, err
	}
	if err := d.interactionDispatcher.LearnGlobalCommands(ac.GlobalCommands()); err != nil {
		return d, err
	}

	d.debugHandler, err = commands.ConfigDebugHandler(d)
	if err != nil {
		return d, err
	}

	d.reactionHandler = commands.NewReactionHandler(d)

	d.discordMsgHandler = dispatcher.NewDispatcher(d)

	d.msgHandlers = msghandler.NewHandlers(d, msghandler.Options{
		DefaultCommandIndicator: "!",
		ErrorColor:              0xff0000,
		SuccessColor:            0xaa63ff,
	})

	if err := d.interactionDispatcher.LearnGlobalCommands(uc.GlobalCommands()); err != nil {
		return d, err
	}

	if err := d.interactionDispatcher.LearnGlobalCommands(cc.GlobalCommands()); err != nil {
		return d, err
	}

	if err := d.interactionDispatcher.LearnGlobalCommands(ac.GlobalCommands()); err != nil {
		return d, err
	}

	d.guildCmdGen = func(gid snowflake.Snowflake) ([]cmdhandler.InteractionCommandHandler, error) {
		cmds := make([]cmdhandler.InteractionCommandHandler, 0, 10)

		gcmds, err := uc.GuildCommands(gid)
		if err != nil {
			return nil, errors.Wrap(err, "could not get UserCommands.GuildCommands", "gid", gid)
		}

		cmds = append(cmds, gcmds...)

		gcmds, err = ac.GuildCommands(gid)
		if err != nil {
			return nil, errors.Wrap(err, "could not get AdminCommands.GuildCommands", "gid", gid)
		}

		cmds = append(cmds, gcmds...)

		gcmds, err = cc.GuildCommands(gid)
		if err != nil {
			return nil, errors.Wrap(err, "could not get ConfigCommands.GuildCommands", "gid", gid)
		}

		cmds = append(cmds, gcmds...)

		return cmds, nil
	}

	botConfig := bot.Config{
		ClientID:     conf.ClientID,
		ClientSecret: conf.ClientSecret,
		BotToken:     conf.ClientToken,
		APIURL:       DiscordAPI,
		NumWorkers:   conf.NumWorkers,

		OS:          "linux",
		BotName:     conf.BotName,
		BotPresence: conf.BotPresence,

		UseSlashCommands:    true,
		GlobalSlashCommands: d.interactionDispatcher.GlobalCommands(),
	}

	d.bot = bot.NewDiscordBot(d, botConfig, botPermissions, botIntents)

	return d, nil
}

func (d *dependencies) Close() {
	if d.db != nil {
		d.db.Close() //nolint:errcheck // not needed
	}

	if d.wsClient != nil {
		d.wsClient.Close()
	}
}

func (d *dependencies) SendAllowed() bool                             { return d.sendAllowed }
func (d *dependencies) InteractionSendAllowed() bool                  { return d.interactionSendAllowed }
func (d *dependencies) Logger() Logger                                { return d.logger }
func (d *dependencies) GuildAPI() storage.GuildAPI                    { return d.guildAPI }
func (d *dependencies) TrialAPI() storage.TrialAPI                    { return d.trialAPI }
func (d *dependencies) HTTPDoer() httpclient.Doer                     { return d.httpDoer }
func (d *dependencies) HTTPClient() jsonapi.HTTPClient                { return d.httpClient }
func (d *dependencies) WSDialer() wsclient.Dialer                     { return d.wsDialer }
func (d *dependencies) WSClient() wsapi.WSClient                      { return d.wsClient }
func (d *dependencies) DiscordJSONClient() *jsonapi.DiscordJSONClient { return d.jsClient }
func (d *dependencies) MessageRateLimiter() *rate.Limiter             { return d.messageRateLimiter }
func (d *dependencies) ConnectRateLimiter() *rate.Limiter             { return d.connectRateLimiter }
func (d *dependencies) ReactionsRateLimiter() *rate.Limiter           { return d.reactionsRateLimiter }
func (d *dependencies) BotSession() *session.Session                  { return d.botSession }
func (d *dependencies) CommandHandler() *cmdhandler.CommandHandler    { return d.cmdHandler }
func (d *dependencies) ConfigHandler() *cmdhandler.CommandHandler     { return d.configHandler }
func (d *dependencies) AdminHandler() *cmdhandler.CommandHandler      { return d.adminHandler }
func (d *dependencies) DebugHandler() *cmdhandler.CommandHandler      { return d.debugHandler }
func (d *dependencies) ReactionHandler() reactions.Handler            { return d.reactionHandler }
func (d *dependencies) MessageHandler() msghandler.Handlers           { return d.msgHandlers }
func (d *dependencies) ErrReporter() errreport.Reporter               { return d.rep }
func (d *dependencies) Census() *telemetry.Census                     { return d.census }
func (d *dependencies) Bot() *bot.DiscordBot                          { return d.bot }
func (d *dependencies) StatsHub() *stats.Hub                          { return d.statsHub }
func (d *dependencies) Dispatcher() bot.Dispatcher                    { return d.discordMsgHandler }
func (d *dependencies) PermissionsManager() *permissions.Manager {
	return d.permissionsManager
}

func (d *dependencies) InteractionDispatcher() *cmdhandler.InteractionDispatcher {
	return d.interactionDispatcher
}

func (d *dependencies) MessageHandlerRecorder() *bstats.ActivityRecorder {
	ar, _ := d.statsHub.Get("raw_msgs")
	return ar
}

func (d *dependencies) GuildCommandGenerator() func(snowflake.Snowflake) ([]cmdhandler.InteractionCommandHandler, error) {
	return d.guildCmdGen
}
