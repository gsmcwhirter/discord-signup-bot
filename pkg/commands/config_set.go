package commands

import (
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/logging/level"

	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/logging"
)

type argPair struct {
	key, val string
}

func (c *ConfigCommands) setHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "configCommands.set", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling configCommand", "command", "set", "set_args", msg.Contents())

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	argPairs := make([]argPair, 0, len(msg.Contents()))

	for _, arg := range msg.Contents() {
		if arg == "" {
			continue
		}

		argPairList := strings.SplitN(arg, "=", 2)
		if len(argPairList) != 2 {
			return r, fmt.Errorf("could not parse setting '%s'", arg)
		}

		ap := argPair{
			key: argPairList[0],
		}

		switch strings.ToLower(argPairList[0]) {
		case "adminrole":
			g, ok := c.deps.BotSession().Guild(msg.GuildID())
			if !ok {
				return r, errors.New("could not find guild to look up role")
			}

			parts := strings.Split(argPairList[1], ",")
			rids := make([]string, 0, len(parts))

			for _, rn := range parts {
				rn = strings.TrimSpace(rn)
				if rn == "" {
					continue
				}

				rid, ok := g.RoleWithName(rn)
				if !ok {
					return r, fmt.Errorf("could not find role with name '%s'", rn)
				}

				rids = append(rids, rid.ToString())
			}

			ap.val = strings.Join(rids, ",")

		default:
			ap.val = argPairList[1]
		}

		argPairs = append(argPairs, ap)
	}

	if len(argPairs) == 0 {
		return r, errors.New("no settings to save")
	}

	t, err := c.deps.GuildAPI().NewTransaction(ctx, true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	bGuild, err := t.AddGuild(ctx, msg.GuildID().ToString())
	if err != nil {
		return r, errors.Wrap(err, "unable to find guild")
	}

	s := bGuild.GetSettings(ctx)
	for _, ap := range argPairs {
		err = s.SetSettingString(ctx, ap.key, ap.val)
		if err != nil {
			return r, err
		}
	}
	bGuild.SetSettings(ctx, s)

	err = t.SaveGuild(ctx, bGuild)
	if err != nil {
		return r, errors.Wrap(err, "could not save guild settings")
	}

	err = t.Commit(ctx)
	if err != nil {
		return r, errors.Wrap(err, "could not save guild settings")
	}

	return c.listHandler(cmdhandler.NewWithContents(msg, ""))
}
