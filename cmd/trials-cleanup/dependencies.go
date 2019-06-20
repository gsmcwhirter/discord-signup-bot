package main

import (
	"context"
	"time"

	bolt "github.com/coreos/bbolt"

	log "github.com/gsmcwhirter/go-util/v5/logging"
	census "github.com/gsmcwhirter/go-util/v5/stats"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

type dependencies struct {
	logger   log.Logger
	db       *bolt.DB
	trialAPI storage.TrialAPI
	guildAPI storage.GuildAPI
	// botSession *etfapi.Session
	census *census.Census
}

func createDependencies(conf config) (*dependencies, error) {
	var err error

	d := &dependencies{}
	logger := log.NewLogfmtLogger()
	logger = log.With(logger, "timestamp", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	d.logger = logger

	d.db, err = bolt.Open(conf.Database, 0660, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return d, err
	}

	d.trialAPI, err = storage.NewBoltTrialAPI(d.db, d.census)
	if err != nil {
		return d, err
	}

	d.guildAPI, err = storage.NewBoltGuildAPI(context.Background(), d.db, d.census)
	if err != nil {
		return d, err
	}

	// d.botSession = etfapi.NewSession()

	return d, nil
}

func (d *dependencies) Close() {
	if d.db != nil {
		d.db.Close() // nolint: errcheck
	}
}

func (d *dependencies) Logger() log.Logger {
	return d.logger
}

func (d *dependencies) TrialAPI() storage.TrialAPI {
	return d.trialAPI
}

func (d *dependencies) GuildAPI() storage.GuildAPI {
	return d.guildAPI
}

// func (d *dependencies) BotSession() *etfapi.Session {
// 	return d.botSession
// }
