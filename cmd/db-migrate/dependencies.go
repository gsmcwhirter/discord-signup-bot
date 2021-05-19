package main

import (
	"context"

	bolt "go.etcd.io/bbolt"

	log "github.com/gsmcwhirter/go-util/v8/logging"
	"github.com/gsmcwhirter/go-util/v8/telemetry"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/pgxutil"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

type dependencies struct {
	logger   log.Logger
	db       *bolt.DB
	trialAPI storage.TrialAPI
	guildAPI storage.GuildAPI
	census   *telemetry.Census
	pgpool   *pgxpool.Pool
}

func createDependencies(ctx context.Context, conf config) (*dependencies, error) {
	var err error

	d := &dependencies{}
	logger := log.NewJSONLogger()
	logger = log.With(logger, "timestamp", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	d.logger = logger

	poolConf, err := pgxpool.ParseConfig(conf.Pg)
	if err != nil {
		return d, err
	}

	poolConf.ConnConfig.Logger = &pgxutil.Logger{Logger: d.logger}
	poolConf.ConnConfig.LogLevel = pgx.LogLevelWarn

	d.pgpool, err = pgxpool.ConnectConfig(ctx, poolConf)
	if err != nil {
		return d, err
	}

	d.guildAPI, err = storage.NewPgGuildAPI(ctx, d.pgpool, d.census)
	if err != nil {
		return d, err
	}

	d.trialAPI, err = storage.NewPgTrialAPI(d.pgpool, d.census)
	if err != nil {
		return d, err
	}

	return d, nil
}

func (d *dependencies) Close() {
	if d.db != nil {
		d.db.Close() //nolint:errcheck // not needed
	}

	if d.pgpool != nil {
		d.pgpool.Close()
	}
}

func (d *dependencies) Logger() log.Logger         { return d.logger }
func (d *dependencies) TrialAPI() storage.TrialAPI { return d.trialAPI }
func (d *dependencies) GuildAPI() storage.GuildAPI { return d.guildAPI }
