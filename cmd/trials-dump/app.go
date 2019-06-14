package main

import (
	"fmt"

	"github.com/gsmcwhirter/go-util/v2/deferutil"
	"github.com/pkg/errors"

	"github.com/gsmcwhirter/discord-bot-lib/v6/snowflake"
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

	// uid, err := snowflake.FromString(c.User)
	// if err != nil {
	// 	return errors.Wrap(err, "could not parse user id")
	// }

	gid, err := snowflake.FromString(c.Guild)
	if err != nil {
		return errors.Wrap(err, "could not parse guild id")
	}

	// cid, err := snowflake.FromString(c.Channel)
	// if err != nil {
	// 	return errors.Wrap(err, "could not parse channel id")
	// }

	if c.AllGuilds {
		return dumpAllGuilds(deps)
	}

	if err := dumpGuildSettings(deps, gid); err != nil {
		return err
	}

	if err := dumpGuildTrials(deps, gid); err != nil {
		return err
	}

	return nil
}

func dumpAllGuilds(deps *dependencies) error {
	return nil
}

func dumpGuildSettings(deps *dependencies, gid snowflake.Snowflake) error {
	t, err := deps.GuildAPI().NewTransaction(false)
	if err != nil {
		return errors.Wrap(err, "could not get settings transaction")
	}
	defer deferutil.CheckDefer(t.Rollback)

	g, err := t.GetGuild(gid.ToString())
	if err != nil {
		return errors.Wrap(err, "could not get guild for settings")
	}

	gsettings := g.GetSettings()
	fmt.Printf("%+v\n\n", gsettings)
	return nil
}

func dumpGuildTrials(deps *dependencies, gid snowflake.Snowflake) error {
	t, err := deps.TrialAPI().NewTransaction(gid.ToString(), false)
	if err != nil {
		return errors.Wrap(err, "could not get trials transaction")
	}
	defer deferutil.CheckDefer(t.Rollback)

	for _, t := range t.GetTrials() {
		fmt.Printf(`Name: %s
	State: %s
	SignupChannel: %s
	AnnounceChannel: %s
	Description: %s
	Role Counts:`, t.GetName(), t.GetState(), t.GetSignupChannel(), t.GetAnnounceChannel(), t.GetDescription())
		for _, rc := range t.GetRoleCounts() {
			fmt.Printf(`
		%s: %d`, rc.GetRole(), rc.GetCount())
		}
		fmt.Printf(`
	Signups:`)
		for _, su := range t.GetSignups() {
			fmt.Printf(`
		%s: %s`, su.GetName(), su.GetRole())
		}
		fmt.Println()
		fmt.Println()
	}

	return nil
}
