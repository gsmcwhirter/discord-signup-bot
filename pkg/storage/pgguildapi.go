package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/gsmcwhirter/go-util/v8/errors"
	"github.com/gsmcwhirter/go-util/v8/telemetry"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"google.golang.org/protobuf/proto"
)

// var settingsBucket = []byte("GuildRecords")

type pgGuildAPI struct {
	db     *pgxpool.Pool
	census *telemetry.Census
}

// NewPgGuildAPI constructs a postgres-backed GuildAPI
func NewPgGuildAPI(ctx context.Context, db *pgxpool.Pool, c *telemetry.Census) (GuildAPI, error) {
	_, span := c.StartSpan(ctx, "pgGuildAPI.NewPgGuildAPI")
	defer span.End()

	b := pgGuildAPI{
		db:     db,
		census: c,
	}

	return &b, nil
}

func (p *pgGuildAPI) AllGuilds(ctx context.Context) ([]string, error) {
	_, span := p.census.StartSpan(ctx, "pgGuildAPI.AllGuilds")
	defer span.End()

	var guilds []string

	rs, err := p.db.Query(ctx, `
	SELECT guild_id 
	FROM guild_settings`)

	if err != nil && err != pgx.ErrNoRows {
		return nil, nil
	}
	defer rs.Close()

	var gname string
	for rs.Next() {
		if err := rs.Scan(&gname); err != nil {
			return nil, errors.Wrap(err, "could not scan guild id")
		}

		guilds = append(guilds, gname)
	}

	return guilds, nil
}

func (p *pgGuildAPI) NewTransaction(ctx context.Context, writable bool) (GuildAPITx, error) {
	_, span := p.census.StartSpan(ctx, "pgGuildAPI.NewTransaction")
	defer span.End()

	opts := pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadOnly,
	}

	if writable {
		opts.AccessMode = pgx.ReadWrite
	}

	tx, err := p.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &PgGuildAPITx{
		tx:     tx,
		census: p.census,
	}, nil
}

type PgGuildAPITx struct {
	tx     pgx.Tx
	census *telemetry.Census
}

func (p *PgGuildAPITx) Commit(ctx context.Context) error {
	_, span := p.census.StartSpan(ctx, "PgGuildAPITx.Commit")
	defer span.End()

	return p.tx.Commit(ctx)
}

func (p *PgGuildAPITx) Rollback(ctx context.Context) error {
	_, span := p.census.StartSpan(ctx, "PgGuildAPITx.Rollback")
	defer span.End()

	err := p.tx.Rollback(ctx)
	if err != nil && err != pgx.ErrTxClosed {
		return err
	}
	return nil
}

func (p *PgGuildAPITx) GetGuild(ctx context.Context, name string) (Guild, error) {
	_, span := p.census.StartSpan(ctx, "PgGuildAPITx.GetGuild")
	defer span.End()

	return p.getGuildProto(ctx, name)
}

func (p *PgGuildAPITx) getGuildProto(ctx context.Context, name string) (Guild, error) {
	pGuild := ProtoGuild{}

	r := p.tx.QueryRow(ctx, `
	SELECT settings
	FROM guild_settings WHERE guild_id = $1`, name)

	var val []byte
	if err := r.Scan(&val); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrGuildNotExist
		}
		return nil, errors.Wrap(err, "could not retrieve guild settings")
	}

	err := proto.Unmarshal(val, &pGuild)
	if err != nil {
		return nil, errors.Wrap(err, "guild record is corrupt")
	}

	pGuild.Name = strings.TrimSpace(pGuild.Name)

	return &protoGuild{
		protoGuild: &pGuild,
		census:     p.census,
	}, nil
}

func (p *PgGuildAPITx) GetGuildPg(ctx context.Context, name string) (Guild, error) {
	pGuild := ProtoGuild{}

	r := p.tx.QueryRow(ctx, `
	SELECT guild_id, command_indicator,
		   announce_channel, signup_channel,
		   admin_channel, announce_to,
		   show_after_signup, show_after_withdraw,
		   hide_reactions_announce, hide_reactions_show
	FROM guild_settings WHERE guild_id = $1`, name)

	if err := r.Scan(
		&pGuild.Name, &pGuild.CommandIndicator,
		&pGuild.AnnounceChannel, &pGuild.SignupChannel,
		&pGuild.AdminChannel, &pGuild.AnnounceTo,
		&pGuild.ShowAfterSignup, &pGuild.ShowAfterWithdraw,
		&pGuild.HideReactionsAnnounce, &pGuild.HideReactionsShow,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrGuildNotExist
		}
		return nil, errors.Wrap(err, "could not retrieve guild settings")
	}

	pGuild.Name = strings.TrimSpace(pGuild.Name)

	rs, err := p.tx.Query(ctx, `SELECT admin_role FROM guild_admin_roles WHERE guild_id = $1`, name)
	if err != nil && err != pgx.ErrNoRows {
		return nil, errors.Wrap(err, "could not retrieve guild admin roles")
	}
	defer rs.Close()

	adminRoles := make([]string, 0, 5)
	var role string

	for rs.Next() {
		if err := rs.Scan(&role); err != nil {
			return nil, errors.Wrap(err, "could not retrieve guild admin roles information")
		}
		adminRoles = append(adminRoles, role)
	}

	pGuild.AdminRole = strings.Join(adminRoles, ",")

	return &protoGuild{
		protoGuild: &pGuild,
		census:     p.census,
	}, nil
}

func (p *PgGuildAPITx) AddGuild(ctx context.Context, name string) (Guild, error) {
	_, span := p.census.StartSpan(ctx, "PgGuildAPITx.AddGuild")
	defer span.End()

	guild, err := p.GetGuild(ctx, name)
	if err == ErrGuildNotExist {
		guild = &protoGuild{
			protoGuild: &ProtoGuild{Name: name},
			census:     p.census,
		}
		err = nil
	}
	return guild, err
}

func (p *PgGuildAPITx) getAdminRoles(ctx context.Context, gid string) ([]string, error) {
	var roles []string

	rs, err := p.tx.Query(ctx, `
	SELECT admin_role 
	FROM guild_admin_roles
	WHERE guild_id = $1`, gid)

	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}
	defer rs.Close()

	var ar string
	for rs.Next() {
		if err := rs.Scan(&ar); err != nil {
			return nil, errors.Wrap(err, "could not scan role name")
		}

		roles = append(roles, ar)
	}

	return roles, nil
}

func (p *PgGuildAPITx) saveProtoGuild(ctx context.Context, guild Guild) error {
	gid := guild.GetName(ctx)
	gs := guild.GetSettings(ctx)

	serial, err := guild.Serialize(ctx)
	if err != nil {
		return errors.Wrap(err, "could not serialize guild data")
	}

	_, err = p.tx.Exec(ctx, `
	INSERT INTO guild_settings (guild_id, settings, command_indicator, announce_channel, signup_channel, admin_channel, announce_to, show_after_signup, show_after_withdraw, hide_reactions_announce, hide_reactions_show)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	ON CONFLICT (guild_id) DO UPDATE
	SET 
		settings = EXCLUDED.settings,
		command_indicator = EXCLUDED.command_indicator,
		announce_channel = EXCLUDED.announce_channel,
		signup_channel = EXCLUDED.signup_channel,
		admin_channel = EXCLUDED.admin_channel,
		announce_to = EXCLUDED.announce_to,
		show_after_signup = EXCLUDED.show_after_signup,
		show_after_withdraw = EXCLUDED.show_after_withdraw,
		hide_reactions_announce = EXCLUDED.hide_reactions_announce,
		hide_reactions_show = EXCLUDED.hide_reactions_show
	`, gid, serial, gs.ControlSequence, gs.AnnounceChannel, gs.SignupChannel, gs.AdminChannel, gs.AnnounceTo, gs.ShowAfterSignup, gs.ShowAfterWithdraw, gs.HideReactionsAnnounce, gs.HideReactionsShow)

	if err != nil {
		return errors.Wrap(err, "could not upsert guild_settings")
	}

	existingRoles, err := p.getAdminRoles(ctx, gid)
	if err != nil {
		return errors.Wrap(err, "could not get existing admin roles")
	}

	toInsert, toDelete := diffStringSlices(gs.AdminRoles, existingRoles)

	if len(toInsert) > 0 {
		insertArgs := make([]interface{}, len(toInsert)+1)
		insertArgs[0] = gid
		for i, v := range toInsert {
			insertArgs[i+1] = v
		}

		_, err = p.tx.Exec(ctx, fmt.Sprintf(`
		INSERT INTO guild_admin_roles (guild_id, admin_role)
		VALUES %s
		`, genPlaceholders("($1, %s)", ", ", 2, len(toInsert))), insertArgs...)

		if err != nil {
			return errors.Wrap(err, "could not insert new admin roles")
		}
	}

	if len(toDelete) > 0 {
		deleteArgs := make([]interface{}, len(toDelete)+1)
		deleteArgs[0] = gid
		for i, v := range toDelete {
			deleteArgs[i+1] = v
		}

		res, err := p.tx.Exec(ctx, fmt.Sprintf(`
		DELETE FROM guild_admin_roles
		WHERE guild_id = $1
		AND admin_role IN (%s)
		`, genPlaceholders("%s", ", ", 2, len(toDelete))), deleteArgs...)
		if err != nil {
			return errors.Wrap(err, "could not delete old admin roles")
		}

		if res.RowsAffected() != int64(len(toDelete)) {
			return errors.Wrap(ErrTooManyRows, "could not delete old admin roles")
		}
	}

	return nil
}

func (p *PgGuildAPITx) SaveGuild(ctx context.Context, guild Guild) error {
	ctx, span := p.census.StartSpan(ctx, "PgGuildAPITx.SaveGuild")
	defer span.End()

	return p.saveProtoGuild(ctx, guild)
}
