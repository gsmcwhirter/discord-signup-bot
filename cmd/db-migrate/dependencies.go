package main

import (
	"context"
	"time"

	bolt "go.etcd.io/bbolt"

	log "github.com/gsmcwhirter/go-util/v7/logging"
	"github.com/gsmcwhirter/go-util/v7/telemetry"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/pgxutil"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

type dependencies struct {
	logger      log.Logger
	db          *bolt.DB
	oldTrialAPI storage.TrialAPI
	oldGuildAPI storage.GuildAPI
	census      *telemetry.Census
	pgpool      *pgxpool.Pool
	newTrialAPI storage.TrialAPI
	newGuildAPI storage.GuildAPI
}

func createDependencies(ctx context.Context, conf config) (*dependencies, error) {
	var err error

	d := &dependencies{}
	logger := log.NewJSONLogger()
	logger = log.With(logger, "timestamp", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	d.logger = logger

	d.db, err = bolt.Open(conf.Database, 0o660, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return d, err
	}

	d.oldTrialAPI, err = storage.NewBoltTrialAPI(d.db, d.census)
	if err != nil {
		return d, err
	}

	d.oldGuildAPI, err = storage.NewBoltGuildAPI(ctx, d.db, d.census)
	if err != nil {
		return d, err
	}

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

	d.newGuildAPI, err = storage.NewPgGuildAPI(ctx, d.pgpool, d.census)
	if err != nil {
		return d, err
	}

	d.newTrialAPI, err = storage.NewPgTrialAPI(d.pgpool, d.census)
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

func (d *dependencies) Logger() log.Logger            { return d.logger }
func (d *dependencies) OldTrialAPI() storage.TrialAPI { return d.oldTrialAPI }
func (d *dependencies) OldGuildAPI() storage.GuildAPI { return d.oldGuildAPI }
func (d *dependencies) NewTrialAPI() storage.TrialAPI { return d.newTrialAPI }
func (d *dependencies) NewGuildAPI() storage.GuildAPI { return d.newGuildAPI }
