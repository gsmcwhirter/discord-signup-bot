package main

import (
	"fmt"

	"github.com/gsmcwhirter/discord-bot-lib/v8/snowflake"
	"github.com/gsmcwhirter/go-util/v3/deferutil"
	"github.com/gsmcwhirter/go-util/v3/errors"
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
	tx, err := deps.TrialAPI().NewTransaction(gid.ToString(), true)
	if err != nil {
		return errors.Wrap(err, "could not get trials transaction")
	}
	defer deferutil.CheckDefer(tx.Rollback)

	ct := 0
	trials := tx.GetTrials()
	for _, t := range trials {
		if len(t.GetName()) > maxlen {
			fmt.Printf("Deleting `%s`\n", t.GetName())
			if err := tx.DeleteTrial(t.GetName()); err != nil {
				return errors.Wrap(err, "could not delete trial")
			}
			ct++
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "could not commit deletes")
	}

	fmt.Printf("Deleted %d trials\n", ct)

	return nil
}
