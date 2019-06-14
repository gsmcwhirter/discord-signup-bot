package storage

import (
	"github.com/gsmcwhirter/go-util/v3/deferutil"
	"github.com/gsmcwhirter/go-util/v3/errors"

	"github.com/gsmcwhirter/discord-bot-lib/v7/snowflake"
)

// GetSettings is a wrapper to get the configuration settings for a guild
//
// NOTE: this cannot be called after another transaction has been started
func GetSettings(gapi GuildAPI, gid snowflake.Snowflake) (GuildSettings, error) {
	t, err := gapi.NewTransaction(false)
	if err != nil {
		return GuildSettings{}, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	bGuild, err := t.AddGuild(gid.ToString())
	if err != nil {
		return GuildSettings{}, errors.Wrap(err, "unable to find guild")
	}

	return bGuild.GetSettings(), nil
}
