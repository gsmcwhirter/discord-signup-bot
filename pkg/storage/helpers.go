package storage

import (
	"context"
	"strconv"
	"strings"

	"github.com/gsmcwhirter/go-util/v7/deferutil"
	"github.com/gsmcwhirter/go-util/v7/errors"

	"github.com/gsmcwhirter/discord-bot-lib/v13/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v13/snowflake"
)

// GetSettings is a wrapper to get the configuration settings for a guild
//
// NOTE: this cannot be called after another transaction has been started
func GetSettings(ctx context.Context, gapi GuildAPI, gid snowflake.Snowflake) (GuildSettings, error) {
	t, err := gapi.NewTransaction(ctx, false)
	if err != nil {
		return GuildSettings{}, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	bGuild, err := t.AddGuild(ctx, gid.ToString())
	if err != nil {
		return GuildSettings{}, errors.Wrap(err, "unable to find guild")
	}

	return bGuild.GetSettings(ctx), nil
}

func userMentionOverflowFix(userMention string) string {
	if !strings.HasPrefix(userMention, "<@!-") {
		return userMention
	}

	var i int64
	var err error
	if i, err = strconv.ParseInt(userMention[3:len(userMention)-1], 10, 64); err != nil {
		return userMention
	}

	return cmdhandler.UserMentionString(snowflake.Snowflake(uint64(i)))
}
