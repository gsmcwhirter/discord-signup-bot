package main

import (
	"context"
	"fmt"

	"github.com/gsmcwhirter/discord-bot-lib/v15/snowflake"
	"github.com/gsmcwhirter/go-util/v7/deferutil"
	"github.com/gsmcwhirter/go-util/v7/errors"
)

type config struct {
	Database string `mapstructure:"database"`
	// User      string `mapstructure:"user"`
	Guild string `mapstructure:"guild"`
	// Channel   string `mapstructure:"channel"`
	AllGuilds bool `mapstructure:"all_guilds"`
}

func start(c config) error {
	fmt.Printf("%+v\n", c)

	deps, err := createDependencies(c)
	if err != nil {
		return err
	}
	defer deps.Close()

	gid, err := snowflake.FromString(c.Guild)
	if err != nil {
		return errors.Wrap(err, "could not parse guild id")
	}

	if err := cleanupGuildTrials(deps, gid, 200); err != nil {
		return err
	}

	return nil
}

func cleanupGuildTrials(deps *dependencies, gid snowflake.Snowflake, maxlen int) error {
	ctx := context.Background()

	tx, err := deps.TrialAPI().NewTransaction(ctx, gid.ToString(), true)
	if err != nil {
		return errors.Wrap(err, "could not get trials transaction")
	}
	defer deferutil.CheckDefer(func() error { return tx.Rollback(ctx) })

	ct := 0
	trials := tx.GetTrials(ctx)
	for _, t := range trials {
		tName := t.GetName(ctx)
		if len(tName) > maxlen {
			fmt.Printf("Deleting `%s`\n", tName)
			if err := tx.DeleteTrial(ctx, tName); err != nil {
				return errors.Wrap(err, "could not delete trial")
			}
			ct++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return errors.Wrap(err, "could not commit deletes")
	}

	fmt.Printf("Deleted %d trials\n", ct)

	return nil
}
