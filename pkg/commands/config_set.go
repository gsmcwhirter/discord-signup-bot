package commands

import (
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v7/deferutil"
	"github.com/gsmcwhirter/go-util/v7/errors"
	"github.com/gsmcwhirter/go-util/v7/logging/level"

	"github.com/gsmcwhirter/discord-bot-lib/v16/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v16/logging"
)

type argPair struct {
	key, val string
}

func (c *configCommands) set(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "configCommands.set", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

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
			rid, ok := g.RoleWithName(argPairList[1])
			if !ok {
				return r, fmt.Errorf("could not find role with name '%s'", argPairList[1])
			}

			ap.val = rid.ToString()
		default:
			ap.val = argPairList[1]
		}

		argPairs = append(argPairs, ap)
	}

	if len(argPairs) == 0 {
		return r, errors.New("no settings to save")
	}

	t, err := c.deps.GuildAPI().NewTransaction(msg.Context(), true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(msg.Context()) })

	bGuild, err := t.AddGuild(msg.Context(), msg.GuildID().ToString())
	if err != nil {
		return r, errors.Wrap(err, "unable to find guild")
	}

	s := bGuild.GetSettings(msg.Context())
	for _, ap := range argPairs {
		err = s.SetSettingString(msg.Context(), ap.key, ap.val)
		if err != nil {
			return r, err
		}
	}
	bGuild.SetSettings(msg.Context(), s)

	err = t.SaveGuild(msg.Context(), bGuild)
	if err != nil {
		return r, errors.Wrap(err, "could not save guild settings")
	}

	err = t.Commit(msg.Context())
	if err != nil {
		return r, errors.Wrap(err, "could not save guild settings")
	}

	return c.list(cmdhandler.NewWithContents(msg, ""))
}
