package commands

import (
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v3/deferutil"
	"github.com/gsmcwhirter/go-util/v3/errors"
	"github.com/gsmcwhirter/go-util/v3/logging/level"

	"github.com/gsmcwhirter/discord-bot-lib/v7/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v7/logging"
)

type argPair struct {
	key, val string
}

func (c *configCommands) set(msg cmdhandler.Message) (cmdhandler.Response, error) {
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

	t, err := c.deps.GuildAPI().NewTransaction(true)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	bGuild, err := t.AddGuild(msg.GuildID().ToString())
	if err != nil {
		return r, errors.Wrap(err, "unable to find guild")
	}

	s := bGuild.GetSettings()
	for _, ap := range argPairs {
		err = s.SetSettingString(ap.key, ap.val)
		if err != nil {
			return r, err
		}
	}
	bGuild.SetSettings(s)

	err = t.SaveGuild(bGuild)
	if err != nil {
		return r, errors.Wrap(err, "could not save guild settings")
	}

	err = t.Commit()
	if err != nil {
		return r, errors.Wrap(err, "could not save guild settings")
	}

	return c.list(cmdhandler.NewWithContents(msg, ""))
}
