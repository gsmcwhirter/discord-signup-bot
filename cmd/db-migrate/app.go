package main

import (
	"context"
	"fmt"

	"github.com/gsmcwhirter/go-util/v7/deferutil"
	"github.com/gsmcwhirter/go-util/v7/errors"
	"github.com/gsmcwhirter/go-util/v7/logging/level"
	"golang.org/x/sync/errgroup"
)

type config struct {
	Database string `mapstructure:"database"`
	Pg       string `mapstructure:"pg"`
}

func start(c config) error {
	fmt.Printf("%+v\n", c)

	ctx := context.Background()

	deps, err := createDependencies(ctx, c)
	if err != nil {
		return err
	}
	defer deps.Close()

	if err := migrateGuildSettings(ctx, deps); err != nil {
		return err
	}

	if err := migrateAllGuildEvents(ctx, deps); err != nil {
		return err
	}

	return nil
}

func migrateGuildSettings(ctx context.Context, deps *dependencies) error {
	guilds, err := deps.OldGuildAPI().AllGuilds(ctx)
	if err != nil {
		return errors.Wrap(err, "could not list all old guilds")
	}

	readTx, err := deps.OldGuildAPI().NewTransaction(ctx, false)
	if err != nil {
		return errors.Wrap(err, "could not start read transation")
	}
	defer deferutil.CheckDefer(func() error { return readTx.Rollback(ctx) })

	writeTx, err := deps.NewGuildAPI().NewTransaction(ctx, true)
	if err != nil {
		return errors.Wrap(err, "could not start write transaction")
	}
	defer deferutil.CheckDefer(func() error { return writeTx.Rollback(ctx) })

	for _, gname := range guilds {
		level.Info(deps.Logger()).Message("migrating guild settings", "guild_id", gname)
		guild, err := readTx.GetGuild(ctx, gname)
		if err != nil {
			level.Error(deps.Logger()).Err("could not retrieve guild settings", err, "guild_id", gname)
			continue
		}

		if err := writeTx.SaveGuild(ctx, guild); err != nil {
			return errors.Wrap(err, "could not save guild settings", "guild_id", gname)
		}
	}

	return writeTx.Commit(ctx)
}

func migrateAllGuildEvents(ctx context.Context, deps *dependencies) error {
	guilds, err := deps.OldGuildAPI().AllGuilds(ctx)
	if err != nil {
		return errors.Wrap(err, "could not list all old guilds")
	}

	eg, ctx := errgroup.WithContext(ctx)

	for _, gname := range guilds {
		gname := gname
		eg.Go(func() error {
			return errors.Wrap(migrateOneGuildEvents(ctx, deps, gname), "could not migrate events for guild", "guild_id", gname)
		})
	}

	return eg.Wait()
}

func migrateOneGuildEvents(ctx context.Context, deps *dependencies, guildID string) error {
	readTx, err := deps.OldTrialAPI().NewTransaction(ctx, guildID, false)
	if err != nil {
		return errors.Wrap(err, "could not start read transation")
	}
	defer deferutil.CheckDefer(func() error { return readTx.Rollback(ctx) })

	writeTx, err := deps.NewTrialAPI().NewTransaction(ctx, guildID, true)
	if err != nil {
		return errors.Wrap(err, "could not start write transaction")
	}
	defer deferutil.CheckDefer(func() error { return writeTx.Rollback(ctx) })

	events := readTx.GetTrials(ctx)
	for _, event := range events {
		eventName := event.GetName(ctx)
		if len(eventName) > 255 {
			level.Error(deps.Logger()).Err("event name is too long; skipping", err, "guild_id", guildID, "event_name", eventName)
			continue
		}
		if err := writeTx.SaveTrial(ctx, event); err != nil {
			return errors.Wrap(err, "could not save event", "event_name", event.GetName(ctx))
		}
	}

	return writeTx.Commit(ctx)
}
