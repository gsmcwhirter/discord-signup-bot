package commands

import (
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/logging/level"

	"github.com/gsmcwhirter/discord-bot-lib/v19/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v19/logging"
)

func (c *configCommands) get(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "configCommands.get", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

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

	t, err := c.deps.GuildAPI().NewTransaction(msg.Context(), false)
	if err != nil {
		return r, err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(msg.Context()) })

	bGuild, err := t.AddGuild(msg.Context(), msg.GuildID().ToString())
	if err != nil {
		return r, errors.Wrap(err, "unable to find guild")
	}

	s := bGuild.GetSettings(msg.Context())
	sVal, err := s.GetSettingString(msg.Context(), settingName)
	if err != nil {
		return r, fmt.Errorf("'%s' is not the name of a setting", settingName)
	}

	r.Description = fmt.Sprintf("```\n%s: '%s'\n```", settingName, sVal)
	return r, nil
}
