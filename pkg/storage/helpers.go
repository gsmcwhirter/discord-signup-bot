package storage

import (
	"github.com/gsmcwhirter/discord-bot-lib/util"
	"github.com/pkg/errors"
)

// GetSettings TODOC
func GetSettings(gapi GuildAPI, guild string) (s GuildSettings, err error) {
	t, err := gapi.NewTransaction(false)
	if err != nil {
		return
	}
	defer util.CheckDefer(t.Rollback)

	bGuild, err := t.AddGuild(guild)
	if err != nil {
		err = errors.Wrap(err, "unable to find guild")
		return
	}

	s = bGuild.GetSettings()
	return
}
