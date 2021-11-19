package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"

	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
	"github.com/gsmcwhirter/discord-bot-lib/v23/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
)

func (c *ConfigCommands) getInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "configCommands.getInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling config interaction", "command", "get")

	gsettings, err := storage.GetSettings(ctx, c.deps.GuildAPI(), ix.GuildID())
	if err != nil {
		return r, nil, err
	}

	okColor, err := colorToInt(gsettings.MessageColor)
	if err != nil {
		return r, nil, err
	}

	errColor, err := colorToInt(gsettings.ErrorColor)
	if err != nil {
		return r, nil, err
	}

	r.SetColor(errColor)

	if !isAdminChannel(logger, ix, gsettings.AdminChannel, c.deps.BotSession()) {
		level.Info(logger).Message("command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, nil, msghandler.ErrUnauthorized
	}

	var settingName string
	for i := range opts {
		if opts[i].Name == "setting_name" {
			settingName = opts[i].ValueString
			continue
		}
	}

	sVal, err := c.getSettingString(ctx, ix.GuildID(), settingName)
	if err != nil {
		return r, nil, errors.Wrap(err, "could not retrieve setting")
	}

	r.Description = fmt.Sprintf("```\n%s: '%s'\n```", settingName, sVal)
	r.SetColor(okColor)
	return r, nil, nil
}

func (c *ConfigCommands) getHandler(msg cmdhandler.Message) (cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(msg.Context(), "configCommands.getHandler", "guild_id", msg.GuildID().ToString())
	defer span.End()
	msg = cmdhandler.NewWithContext(ctx, msg)

	r := &cmdhandler.SimpleEmbedResponse{
		// To: cmdhandler.UserMentionString(msg.UserID()),
	}

	r.SetReplyTo(msg)

	logger := logging.WithMessage(msg, c.deps.Logger())
	level.Info(logger).Message("handling configCommand", "command", "get", "args", msg.Contents())

	gsettings, err := storage.GetSettings(ctx, c.deps.GuildAPI(), msg.GuildID())
	if err != nil {
		return r, err
	}

	okColor, err := colorToInt(gsettings.MessageColor)
	if err != nil {
		return r, err
	}

	errColor, err := colorToInt(gsettings.ErrorColor)
	if err != nil {
		return r, err
	}

	r.SetColor(errColor)

	if !isAdminChannel(logger, msg, gsettings.AdminChannel, c.deps.BotSession()) {
		level.Info(logger).Message("command not in admin channel", "admin_channel", gsettings.AdminChannel)
		return r, msghandler.ErrUnauthorized
	}

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

	sVal, err := c.getSettingString(ctx, msg.GuildID(), settingName)
	if err != nil {
		return r, errors.Wrap(err, "could not retrieve setting")
	}

	r.Description = fmt.Sprintf("```\n%s: '%s'\n```", settingName, sVal)
	r.SetColor(okColor)
	return r, nil
}

func (c *ConfigCommands) getSettingString(ctx context.Context, gid snowflake.Snowflake, settingName string) (string, error) {
	ctx, span := c.deps.Census().StartSpan(ctx, "configCommands.getSettingString", "guild_id", gid.ToString())
	defer span.End()

	t, err := c.deps.GuildAPI().NewTransaction(ctx, false)
	if err != nil {
		return "", err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	bGuild, err := t.AddGuild(ctx, gid.ToString())
	if err != nil {
		return "", errors.Wrap(err, "unable to find guild")
	}

	s := bGuild.GetSettings(ctx)
	sVal, err := s.GetSettingString(ctx, settingName)
	if err != nil {
		return "", fmt.Errorf("'%s' is not the name of a setting", settingName)
	}

	return sVal, nil
}
