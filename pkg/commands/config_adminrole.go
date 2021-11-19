package commands

import (
	"context"
	"strings"

	"github.com/gsmcwhirter/discord-bot-lib/v23/cmdhandler"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
	"github.com/gsmcwhirter/discord-bot-lib/v23/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/logging/level"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/msghandler"
	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

func (c *ConfigCommands) adminroleAddInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "configCommands.adminroleAddInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling config interaction", "command", "adminrole add")

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

	exists := map[string]bool{}
	for _, rname := range gsettings.AdminRoles {
		exists[rname] = true
	}

	var rid snowflake.Snowflake
	for i := range opts {
		if opts[i].Name != "role" {
			continue
		}

		rid = opts[i].ValueRole
	}

	if exists[rid.ToString()] {
		return r, nil, errors.New("role is already an admin")
	}

	roles := append(gsettings.AdminRoles, rid.ToString()) //nolint:gocritic // this is safe in this case

	if err := c.saveAdminroles(ctx, ix.GuildID(), roles); err != nil {
		return r, nil, errors.Wrap(err, "could not save admin roles")
	}

	r.Description = "Admin role added successfully."
	r.SetColor(okColor)

	return r, nil, nil
}

func (c *ConfigCommands) adminroleListInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "configCommands.adminroleListInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling config interaction", "command", "adminrole list")

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

	roles := make([]string, len(gsettings.AdminRoles))
	for _, ridStr := range gsettings.AdminRoles {
		rid, err := snowflake.FromString(ridStr)
		if err != nil {
			return r, nil, errors.Wrap(err, "could not convert role id to snowflake", "rid", ridStr)
		}

		roles = append(roles, cmdhandler.RoleMentionString(rid))
	}

	r.Title = "Administrator Roles"
	r.Description = strings.Join(roles, "\n")
	r.SetColor(okColor)

	return r, nil, nil
}

func (c *ConfigCommands) adminroleRefreshInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "configCommands.adminroleRefreshInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling config interaction", "command", "adminrole refresh")

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

	if err := c.deps.PermissionsManager().RefreshPermissions(ctx, c.deps.Bot().Config().ClientID, ix.GuildID()); err != nil {
		return r, nil, errors.Wrap(err, "could not refresh permissions")
	}

	r.Description = "Command permissions refreshed successfully."
	r.SetColor(okColor)

	return r, nil, nil
}

func (c *ConfigCommands) adminroleRemoveInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "configCommands.adminroleRemoveInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling config interaction", "command", "adminrole remove")

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

	exists := map[string]bool{}
	for _, rname := range gsettings.AdminRoles {
		exists[rname] = true
	}

	var rid snowflake.Snowflake
	for i := range opts {
		if opts[i].Name != "role" {
			continue
		}

		rid = opts[i].ValueRole
	}

	if !exists[rid.ToString()] {
		return r, nil, errors.New("role is not an admin")
	}

	roles := make([]string, 0, len(gsettings.AdminRoles)-1)
	for _, rname := range gsettings.AdminRoles {
		if rname == rid.ToString() {
			continue
		}

		roles = append(roles, rname)
	}

	if err := c.saveAdminroles(ctx, ix.GuildID(), roles); err != nil {
		return r, nil, errors.Wrap(err, "could not save admin roles")
	}

	r.Description = "Admin role removed successfully."
	r.SetColor(okColor)

	return r, nil, nil
}

func (c *ConfigCommands) adminroleClearInteraction(ix *cmdhandler.Interaction, opts []entity.ApplicationCommandInteractionOption) (cmdhandler.Response, []cmdhandler.Response, error) {
	ctx, span := c.deps.Census().StartSpan(ix.Context(), "configCommands.adminroleClearInteraction", "guild_id", ix.GuildID().ToString())
	defer span.End()

	r := &cmdhandler.SimpleEmbedResponse{}

	logger := logging.WithMessage(ix, c.deps.Logger())
	level.Info(logger).Message("handling config interaction", "command", "adminrole clear")

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

	if err := c.saveAdminroles(ctx, ix.GuildID(), []string{}); err != nil {
		return r, nil, errors.Wrap(err, "could not save admin roles")
	}

	r.Description = "Admin roles cleared successfully."
	r.SetColor(okColor)

	return r, nil, nil
}

func (c *ConfigCommands) saveAdminroles(ctx context.Context, gid snowflake.Snowflake, newRoles []string) error {
	t, err := c.deps.GuildAPI().NewTransaction(ctx, true)
	if err != nil {
		return err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	bGuild, err := t.AddGuild(ctx, gid.ToString())
	if err != nil {
		return errors.Wrap(err, "unable to find guild")
	}

	s := bGuild.GetSettings(ctx)
	s.AdminRoles = newRoles
	bGuild.SetSettings(ctx, s)

	err = t.SaveGuild(ctx, bGuild)
	if err != nil {
		return errors.Wrap(err, "could not save guild settings")
	}

	err = t.Commit(ctx)
	if err != nil {
		return errors.Wrap(err, "could not save guild settings")
	}

	return errors.Wrap(c.deps.PermissionsManager().RefreshPermissions(ctx, c.deps.Bot().Config().ClientID, gid), "could not refresh command permissions")
}
