package storage

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"

	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
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

func unique(ss []string) []string {
	ret := make([]string, 0, len(ss))
	k := make(map[string]struct{}, len(ss))

	for _, s := range ss {
		if _, ok := k[s]; !ok {
			k[s] = struct{}{}
			ret = append(ret, s)
		}
	}

	return ret
}

func diffStringSlices(a, b []string) ([]string, []string) {
	a = unique(a)
	sort.Strings(a)

	b = unique(b)
	sort.Strings(b)

	aOnly := make([]string, 0, len(a))
	bOnly := make([]string, 0, len(b))

	bi := 0

	for ai := 0; ai < len(a); ai++ {
		if bi >= len(b) {
			aOnly = append(aOnly, a[ai])
			continue
		}

		if a[ai] == b[bi] { // in both
			bi++
			continue
		}

		if a[ai] < b[bi] {
			aOnly = append(aOnly, a[ai])
			continue
		}

		if a[ai] > b[bi] {
			bOnly = append(bOnly, b[bi])
			ai--
			bi++
			continue
		}
	}

	for i := bi; i < len(b); i++ {
		bOnly = append(bOnly, b[i])
	}

	return aOnly, bOnly
}

func genPlaceholders(pat, sep string, start, count int) string {
	pieces := make([]string, 0, count)
	for i := start; i < start+count; i++ {
		pieces = append(pieces, fmt.Sprintf(pat, fmt.Sprintf("$%d", i)))
	}

	return strings.Join(pieces, sep)
}
