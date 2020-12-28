package storage

import (
	"context"

	"github.com/gsmcwhirter/go-util/v7/errors"
	"github.com/gsmcwhirter/go-util/v7/telemetry"
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

	return &pgGuildAPITx{
		tx:     tx,
		census: p.census,
	}, nil
}

type pgGuildAPITx struct {
	tx     pgx.Tx
	census *telemetry.Census
}

func (p *pgGuildAPITx) Commit(ctx context.Context) error {
	_, span := p.census.StartSpan(ctx, "pgGuildAPITx.Commit")
	defer span.End()

	return p.tx.Commit(ctx)
}

func (p *pgGuildAPITx) Rollback(ctx context.Context) error {
	_, span := p.census.StartSpan(ctx, "pgGuildAPITx.Rollback")
	defer span.End()

	err := p.tx.Rollback(ctx)
	if err != nil && err != pgx.ErrTxClosed {
		return err
	}
	return nil
}

func (p *pgGuildAPITx) GetGuild(ctx context.Context, name string) (Guild, error) {
	_, span := p.census.StartSpan(ctx, "pgGuildAPITx.GetGuild")
	defer span.End()

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

	pGuild := ProtoGuild{}
	err := proto.Unmarshal(val, &pGuild)
	if err != nil {
		return nil, errors.Wrap(err, "guild record is corrupt")
	}

	return &protoGuild{
		protoGuild: &pGuild,
		census:     p.census,
	}, nil
}

func (p *pgGuildAPITx) AddGuild(ctx context.Context, name string) (Guild, error) {
	_, span := p.census.StartSpan(ctx, "pgGuildAPITx.AddGuild")
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

func (p *pgGuildAPITx) SaveGuild(ctx context.Context, guild Guild) error {
	_, span := p.census.StartSpan(ctx, "pgGuildAPITx.SaveGuild")
	defer span.End()

	serial, err := guild.Serialize(ctx)
	if err != nil {
		return err
	}

	_, err = p.tx.Exec(ctx, `
	INSERT INTO guild_settings (guild_id, settings)
	VALUES ($1, $2)
	ON CONFLICT (guild_id) DO UPDATE
	SET settings = EXCLUDED.settings`, guild.GetName(ctx), serial)

	return err
}
