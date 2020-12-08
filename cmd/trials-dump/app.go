package main

import (
	"context"
	"fmt"

	"github.com/gsmcwhirter/go-util/v7/deferutil"
	"github.com/gsmcwhirter/go-util/v7/errors"

	"github.com/gsmcwhirter/discord-bot-lib/v18/snowflake"
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

	if gid == snowflake.Snowflake(558789102510800917) {
		if err := setGuildAdminRole(deps, gid, "584957583836839936"); err != nil {
			return err
		}
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
	ctx := context.Background()

	t, err := deps.GuildAPI().NewTransaction(ctx, false)
	if err != nil {
		return errors.Wrap(err, "could not get settings transaction")
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	g, err := t.GetGuild(ctx, gid.ToString())
	if err != nil {
		return errors.Wrap(err, "could not get guild for settings")
	}

	gsettings := g.GetSettings(ctx)
	fmt.Printf("SETTINGS: %+v\n\n", gsettings)
	return nil
}

func setGuildAdminRole(deps *dependencies, gid snowflake.Snowflake, roleName string) error {
	ctx := context.Background()

	t, err := deps.GuildAPI().NewTransaction(ctx, true)
	if err != nil {
		return errors.Wrap(err, "could not get settings transaction")
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	g, err := t.GetGuild(ctx, gid.ToString())
	if err != nil {
		return errors.Wrap(err, "could not get guild for settings")
	}

	gsettings := g.GetSettings(ctx)
	gsettings.AdminRole = roleName
	g.SetSettings(ctx, gsettings)

	if err := t.SaveGuild(ctx, g); err != nil {
		return errors.Wrap(err, "could not save guild")
	}

	if err := t.Commit(ctx); err != nil {
		return errors.Wrap(err, "could not commit")
	}

	return nil
}

func dumpGuildTrials(deps *dependencies, gid snowflake.Snowflake) error {
	ctx := context.Background()
	t, err := deps.TrialAPI().NewTransaction(ctx, gid.ToString(), false)
	if err != nil {
		return errors.Wrap(err, "could not get trials transaction")
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	for _, t := range t.GetTrials(ctx) {
		fmt.Printf(`Name: %s
	State: %s
	SignupChannel: %s
	AnnounceChannel: %s
	Description: %s
	Role Counts:`, t.GetName(ctx), t.GetState(ctx), t.GetSignupChannel(ctx), t.GetAnnounceChannel(ctx), t.GetDescription(ctx))
		for _, rc := range t.GetRoleCounts(ctx) {
			fmt.Printf(`
		%s: %d`, rc.GetRole(ctx), rc.GetCount(ctx))
		}
		fmt.Printf(`
	Signups:`)
		for _, su := range t.GetSignups(ctx) {
			fmt.Printf(`
		%s: %s`, su.GetName(ctx), su.GetRole(ctx))
		}
		fmt.Println()
		fmt.Println()
	}

	return nil
}
