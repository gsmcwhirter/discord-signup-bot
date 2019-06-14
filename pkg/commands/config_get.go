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

func (c *configCommands) get(msg cmdhandler.Message) (cmdhandler.Response, error) {
	r := &cmdhandler.SimpleEmbedResponse{
		To: cmdhandler.UserMentionString(msg.UserID()),
	}

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling adminCommand", "command", "get", "args", msg.Contents())

	if msg.ContentErr() != nil {
		return r, msg.ContentErr()
	}

	if len(msg.Contents()) < 1 {
		return r, errors.New("missing setting name")
	}

	if len(msg.Contents()) > 1 {
		return r, errors.New("too many arguments")
	}

	settingName := strings.TrimSpace(msg.Contents()[0])

	t, err := c.deps.GuildAPI().NewTransaction(false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(t.Rollback)

	bGuild, err := t.AddGuild(msg.GuildID().ToString())
	if err != nil {
		return r, errors.Wrap(err, "unable to find guild")
	}

	s := bGuild.GetSettings()
	sVal, err := s.GetSettingString(settingName)
	if err != nil {
		return r, fmt.Errorf("'%s' is not the name of a setting", settingName)
	}

	r.Description = fmt.Sprintf("```\n%s: '%s'\n```", settingName, sVal)
	return r, nil
}
