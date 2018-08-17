package storage

import (
	"github.com/gsmcwhirter/go-util/deferutil"
	"github.com/pkg/errors"
)

// GetSettings is a wrapper to get the configuration settings for a guild
//
// NOTE: this cannot be called after another transaction has been started
func GetSettings(gapi GuildAPI, guild string) (s GuildSettings, err error) {
	t, err := gapi.NewTransaction(false)
	if err != nil {
		return
	}
	defer deferutil.CheckDefer(t.Rollback)

	bGuild, err := t.AddGuild(guild)
	if err != nil {
		err = errors.Wrap(err, "unable to find guild")
		return
	}

	s = bGuild.GetSettings()
	return
}
