package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/logging/level"
	multierror "github.com/hashicorp/go-multierror"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

var errMismatch = errors.New("settings mismatch")

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

	if err := migrateGuildSettings2(ctx, deps); err != nil {
		return err
	}

	// if err := migrateAllGuildEvents2(ctx, deps); err != nil {
	// 	return err
	// }

	if err := checkGuildSettings2(ctx, deps); err != nil {
		return err
	}

	return nil
}

func migrateGuildSettings2(ctx context.Context, deps *dependencies) error {
	guilds, err := deps.GuildAPI().AllGuilds(ctx)
	if err != nil {
		return errors.Wrap(err, "could not list all guilds")
	}

	tx, err := deps.GuildAPI().NewTransaction(ctx, true)
	if err != nil {
		return errors.Wrap(err, "could not start write transaction")
	}
	defer deferutil.CheckDefer(func() error { return tx.Rollback(ctx) })

	for _, gname := range guilds {
		level.Info(deps.Logger()).Message("migrating guild settings", "guild_id", gname)
		guild, err := tx.GetGuild(ctx, gname)
		if err != nil {
			level.Error(deps.Logger()).Err("could not retrieve guild settings", err, "guild_id", gname)
			continue
		}

		if err := tx.SaveGuild(ctx, guild); err != nil {
			return errors.Wrap(err, "could not save guild settings", "guild_id", gname)
		}
	}

	return tx.Commit(ctx)
}

func checkGuildSettings2(ctx context.Context, deps *dependencies) error {
	guilds, err := deps.GuildAPI().AllGuilds(ctx)
	if err != nil {
		return errors.Wrap(err, "could not list all guilds")
	}

	tx, err := deps.GuildAPI().NewTransaction(ctx, false)
	if err != nil {
		return errors.Wrap(err, "could not start read transaction")
	}
	defer deferutil.CheckDefer(func() error { return tx.Rollback(ctx) })

	for _, gname := range guilds {
		// level.Info(deps.Logger()).Message("checking guild settings", "guild_id", gname)
		guild, err := tx.GetGuild(ctx, gname)
		if err != nil {
			level.Error(deps.Logger()).Err("could not retrieve guild settings", err, "guild_id", gname)
			continue
		}

		guild2, err := tx.(*storage.PgGuildAPITx).GetGuildPg(ctx, gname)
		if err != nil {
			level.Error(deps.Logger()).Err("could not retrieve guild settings 2", err, "guild_id", gname)
		}

		if guild.GetName(ctx) != guild2.GetName(ctx) {
			err = multierror.Append(err, errors.Wrap(errMismatch, "name mismatch", "proto", fmt.Sprintf("%q", guild.GetName(ctx)), "pg", fmt.Sprintf("%q", guild2.GetName(ctx))))
		}

		settings := guild.GetSettings(ctx)
		settings2 := guild2.GetSettings(ctx)

		if settings.AdminChannel != settings2.AdminChannel {
			err = multierror.Append(err, errors.Wrap(errMismatch, "admin channel mismatch", "proto", settings.AdminChannel, "pg", settings2.AdminChannel))
		}

		sort.Strings(settings.AdminRoles)
		sort.Strings(settings2.AdminRoles)

		if len(settings.AdminRoles) != len(settings2.AdminRoles) {
			err = multierror.Append(err, errors.Wrap(errMismatch, "admin roles length mismatch", "proto", settings.AdminRoles, "pg", settings2.AdminRoles))
		} else {
			for i := range settings.AdminRoles {
				if settings.AdminRoles[i] != settings2.AdminRoles[i] {
					err = multierror.Append(err, errors.Wrap(errMismatch, "admin roles mismatch", "proto", settings.AdminRoles, "pg", settings2.AdminRoles))
					break
				}
			}
		}

		if settings.AnnounceChannel != settings2.AnnounceChannel {
			err = multierror.Append(err, errors.Wrap(errMismatch, "announce channel mismatch", "proto", settings.AnnounceChannel, "pg", settings2.AnnounceChannel))
		}

		if settings.AnnounceTo != settings2.AnnounceTo {
			err = multierror.Append(err, errors.Wrap(errMismatch, "announce to mismatch", "proto", settings.AnnounceTo, "pg", settings2.AnnounceTo))
		}

		if settings.ControlSequence != settings2.ControlSequence {
			err = multierror.Append(err, errors.Wrap(errMismatch, "control sequence mismatch", "proto", settings.ControlSequence, "pg", settings2.ControlSequence))
		}

		if settings.HideReactionsAnnounce != settings2.HideReactionsAnnounce {
			err = multierror.Append(err, errors.Wrap(errMismatch, "hide reactions announce mismatch", "proto", settings.HideReactionsAnnounce, "pg", settings2.HideReactionsAnnounce))
		}

		if settings.HideReactionsShow != settings2.HideReactionsShow {
			err = multierror.Append(err, errors.Wrap(errMismatch, "hide reactions show mismatch", "proto", settings.HideReactionsShow, "pg", settings2.HideReactionsShow))
		}

		if settings.ShowAfterSignup != settings2.ShowAfterSignup {
			err = multierror.Append(err, errors.Wrap(errMismatch, "show after signup mismatch", "proto", settings.ShowAfterSignup, "pg", settings2.ShowAfterSignup))
		}

		if settings.ShowAfterWithdraw != settings2.ShowAfterWithdraw {
			err = multierror.Append(err, errors.Wrap(errMismatch, "show after withdraw mismatch", "proto", settings.ShowAfterWithdraw, "pg", settings2.ShowAfterWithdraw))
		}

		if settings.SignupChannel != settings2.SignupChannel {
			err = multierror.Append(err, errors.Wrap(errMismatch, "signup channel mismatch", "proto", settings.SignupChannel, "pg", settings2.SignupChannel))
		}

		if err != nil {
			level.Error(deps.Logger()).Err("guild settings mismatch", err, "guild_id", gname)
			continue
		}
	}

	return tx.Commit(ctx)
}

// func migrateGuildSettings(ctx context.Context, deps *dependencies) error {
// 	guilds, err := deps.OldGuildAPI().AllGuilds(ctx)
// 	if err != nil {
// 		return errors.Wrap(err, "could not list all old guilds")
// 	}

// 	readTx, err := deps.OldGuildAPI().NewTransaction(ctx, false)
// 	if err != nil {
// 		return errors.Wrap(err, "could not start read transation")
// 	}
// 	defer deferutil.CheckDefer(func() error { return readTx.Rollback(ctx) })

// 	writeTx, err := deps.NewGuildAPI().NewTransaction(ctx, true)
// 	if err != nil {
// 		return errors.Wrap(err, "could not start write transaction")
// 	}
// 	defer deferutil.CheckDefer(func() error { return writeTx.Rollback(ctx) })

// 	for _, gname := range guilds {
// 		level.Info(deps.Logger()).Message("migrating guild settings", "guild_id", gname)
// 		guild, err := readTx.GetGuild(ctx, gname)
// 		if err != nil {
// 			level.Error(deps.Logger()).Err("could not retrieve guild settings", err, "guild_id", gname)
// 			continue
// 		}

// 		if err := writeTx.SaveGuild(ctx, guild); err != nil {
// 			return errors.Wrap(err, "could not save guild settings", "guild_id", gname)
// 		}
// 	}

// 	return writeTx.Commit(ctx)
// }

// func migrateAllGuildEvents(ctx context.Context, deps *dependencies) error {
// 	guilds, err := deps.OldGuildAPI().AllGuilds(ctx)
// 	if err != nil {
// 		return errors.Wrap(err, "could not list all old guilds")
// 	}

// 	eg, ctx := errgroup.WithContext(ctx)

// 	for _, gname := range guilds {
// 		gname := gname
// 		eg.Go(func() error {
// 			return errors.Wrap(migrateOneGuildEvents(ctx, deps, gname), "could not migrate events for guild", "guild_id", gname)
// 		})
// 	}

// 	return eg.Wait()
// }

// func migrateOneGuildEvents(ctx context.Context, deps *dependencies, guildID string) error {
// 	level.Info(deps.Logger()).Message("migrating events for guild", "guild_id", guildID)

// 	readTx, err := deps.OldTrialAPI().NewTransaction(ctx, guildID, false)
// 	if err != nil {
// 		return errors.Wrap(err, "could not start read transation")
// 	}
// 	defer deferutil.CheckDefer(func() error { return readTx.Rollback(ctx) })

// 	writeTx, err := deps.NewTrialAPI().NewTransaction(ctx, guildID, true)
// 	if err != nil {
// 		return errors.Wrap(err, "could not start write transaction")
// 	}
// 	defer deferutil.CheckDefer(func() error { return writeTx.Rollback(ctx) })

// 	events := readTx.GetTrials(ctx)
// 	for _, event := range events {
// 		eventName := event.GetName(ctx)
// 		level.Info(deps.Logger()).Message("migrating event", "guild_id", guildID, "event_name", eventName)
// 		if len(eventName) > 255 {
// 			level.Error(deps.Logger()).Err("event name is too long; skipping", err, "guild_id", guildID, "event_name", eventName)
// 			continue
// 		}
// 		if err := writeTx.SaveTrial(ctx, event); err != nil {
// 			return errors.Wrap(err, "could not save event", "event_name", event.GetName(ctx))
// 		}
// 	}

// 	return writeTx.Commit(ctx)
// }

// func migrateAllGuildEvents2(ctx context.Context, deps *dependencies) error {
// 	// guilds, err := deps.GuildAPI().AllGuilds(ctx)
// 	// if err != nil {
// 	// 	return errors.Wrap(err, "could not list all old guilds")
// 	// }

// 	guilds := []string{"468646871133454357"}

// 	eg, ctx := errgroup.WithContext(ctx)

// 	for _, gname := range guilds {
// 		gname := gname
// 		eg.Go(func() error {
// 			return errors.Wrap(migrateOneGuildEvents2(ctx, deps, gname), "could not migrate events for guild", "guild_id", gname)
// 		})
// 	}

// 	return eg.Wait()
// }

// func migrateOneGuildEvents2(ctx context.Context, deps *dependencies, guildID string) error {
// 	level.Info(deps.Logger()).Message("migrating events for guild", "guild_id", guildID)

// 	tx, err := deps.TrialAPI().NewTransaction(ctx, guildID, true)
// 	if err != nil {
// 		return errors.Wrap(err, "could not start transation")
// 	}
// 	defer deferutil.CheckDefer(func() error { return tx.Rollback(ctx) })

// 	events := tx.GetTrials(ctx)
// 	for _, event := range events {
// 		eventName := event.GetName(ctx)
// 		level.Info(deps.Logger()).Message("migrating event", "guild_id", guildID, "event_name", eventName)
// 		if len(eventName) > 255 {
// 			level.Error(deps.Logger()).Err("event name is too long; skipping", err, "guild_id", guildID, "event_name", eventName)
// 			continue
// 		}
// 		if err := tx.SaveTrial(ctx, event); err != nil {
// 			return errors.Wrap(err, "could not save event", "event_name", event.GetName(ctx))
// 		}
// 	}

// 	return tx.Commit(ctx)
// }
