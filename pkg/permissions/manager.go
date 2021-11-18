package permissions

import (
	"context"

	"github.com/gsmcwhirter/discord-bot-lib/v23/bot"
	"github.com/gsmcwhirter/discord-bot-lib/v23/bot/session"
	"github.com/gsmcwhirter/discord-bot-lib/v23/discordapi/entity"
	"github.com/gsmcwhirter/discord-bot-lib/v23/logging"
	"github.com/gsmcwhirter/discord-bot-lib/v23/snowflake"
	"github.com/gsmcwhirter/go-util/v8/deferutil"
	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/logging/level"
	"github.com/gsmcwhirter/go-util/v8/telemetry"

	"github.com/gsmcwhirter/discord-signup-bot/pkg/storage"
)

type Logger = interface {
	Log(keyvals ...interface{}) error
	Message(string, ...interface{})
	Err(string, error, ...interface{})
	Printf(string, ...interface{})
}

type permissionDependencies interface {
	Logger() Logger
	GuildAPI() storage.GuildAPI
	Bot() *bot.DiscordBot
	BotSession() *session.Session
	Census() *telemetry.Census
}

var (
	ErrGuildNotFound = errors.New("guild not found")
	ErrMissingData   = errors.New("missing data")
)

type Manager struct {
	deps               permissionDependencies
	restrictedCommands []string
	guildCommands      map[snowflake.Snowflake]map[string]snowflake.Snowflake // guild_id -> command_name -> command_id
}

func NewManager(deps permissionDependencies, restricted []string) *Manager {
	return &Manager{
		deps:               deps,
		restrictedCommands: restricted,
		guildCommands:      make(map[snowflake.Snowflake]map[string]snowflake.Snowflake),
	}
}

func (m *Manager) SetGuildCommands(gid snowflake.Snowflake, cmds []entity.ApplicationCommand) {
	if m.guildCommands == nil {
		m.guildCommands = make(map[snowflake.Snowflake]map[string]snowflake.Snowflake)
	}

	m.guildCommands[gid] = make(map[string]snowflake.Snowflake, len(cmds))
	for i := range cmds {
		m.guildCommands[gid][cmds[i].Name] = cmds[i].IDSnowflake
	}
}

func (m *Manager) RefreshPermissions(ctx context.Context, aid string, gid snowflake.Snowflake) error {
	ctx, span := m.deps.Census().StartSpan(ctx, "Manager.RefreshPermissions", "guild_id", gid.ToString())
	defer span.End()

	logger := logging.WithContext(ctx, m.deps.Logger())
	level.Info(logger).Message("refreshing permissions", "gid", gid)

	g, ok := m.deps.BotSession().Guild(gid)
	if !ok {
		return errors.WithDetails(ErrGuildNotFound, "gid", gid)
	}

	adminRids := g.AllAdministratorRoleIDs()

	t, err := m.deps.GuildAPI().NewTransaction(ctx, false)
	if err != nil {
		return err
	}
	defer deferutil.CheckDefer(func() error { return t.Rollback(ctx) })

	bGuild, err := t.AddGuild(ctx, gid.ToString())
	if err != nil {
		return errors.Wrap(err, "unable to find guild")
	}

	s := bGuild.GetSettings(ctx)
	if err := t.Rollback(ctx); err != nil {
		return errors.Wrap(err, "could not rollback transaction")
	}

	seen := map[snowflake.Snowflake]bool{}

	perms := make([]entity.ApplicationCommandPermission, 0, len(s.AdminRoles)+len(adminRids))
	for _, rid := range adminRids {
		perms = append(perms, entity.ApplicationCommandPermission{
			IDString:    rid.ToString(),
			IDSnowflake: rid,
			Type:        entity.CommandPermissionRole,
			Permission:  true,
		})
		seen[rid] = true
	}

	for _, role := range s.AdminRoles {
		rid, err := snowflake.FromString(role)
		if err != nil {
			logger.Err("malformed admin role; skipping", err, "gid", gid, "role", role)
			continue
		}

		if seen[rid] {
			continue
		}

		perms = append(perms, entity.ApplicationCommandPermission{
			IDString:    rid.ToString(),
			IDSnowflake: rid,
			Type:        entity.CommandPermissionRole,
			Permission:  true,
		})

		seen[rid] = true
	}

	gcmds, ok := m.guildCommands[gid]
	if !ok {
		return errors.Wrap(ErrMissingData, "no guild commands registered", "gid", gid)
	}

	cmdPerms := make([]entity.ApplicationCommandPermissions, 0, len(m.restrictedCommands))
	for _, cname := range m.restrictedCommands {
		cid, ok := gcmds[cname]
		if !ok {
			return errors.Wrap(ErrMissingData, "no command registered", "gid", gid, "cname", cname)
		}

		cmdPerms = append(cmdPerms, entity.ApplicationCommandPermissions{
			IDString:            cid.ToString(),
			ApplicationIDString: aid,
			GuildIDString:       gid.ToString(),
			Permissions:         perms,
		})
	}

	_, err = m.deps.Bot().API().BulkOverwriteGuildCommandPermissions(ctx, aid, gid, cmdPerms)
	if err != nil {
		return errors.Wrap(err, "could not save guild command permissions")
	}

	return nil
}
