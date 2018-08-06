package msghandler

import (
	"github.com/gsmcwhirter/discord-bot-lib/snowflake"
	"github.com/gsmcwhirter/discord-bot-lib/util"
	"github.com/pkg/errors"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

// GetSettings TODOC
func GetSettings(gapi storage.GuildAPI, gid snowflake.Snowflake) (s storage.GuildSettings, err error) {
	t, err := gapi.NewTransaction(false)
	if err != nil {
		return
	}
	defer util.CheckDefer(t.Rollback)

	bGuild, err := t.AddGuild(gid.ToString())
	if err != nil {
		err = errors.Wrap(err, "unable to find guild")
		return
	}

	s = bGuild.GetSettings()
	return
}
