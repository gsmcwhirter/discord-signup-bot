package storage

import (
	"context"
	"strings"

	"github.com/gsmcwhirter/go-util/v7/errors"
	"github.com/gsmcwhirter/go-util/v7/telemetry"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"google.golang.org/protobuf/proto"
)

type pgTrialAPI struct {
	db     *pgxpool.Pool
	census *telemetry.Census
}

// NewPgTrialAPI constructs a boltDB-backed TrialAPI
func NewPgTrialAPI(db *pgxpool.Pool, c *telemetry.Census) (TrialAPI, error) {
	b := pgTrialAPI{
		db:     db,
		census: c,
	}

	return &b, nil
}

func (p *pgTrialAPI) NewTransaction(ctx context.Context, guild string, writable bool) (TrialAPITx, error) {
	_, span := p.census.StartSpan(ctx, "pgTrialAPI.NewTransaction")
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

	return &pgTrialAPITx{
		guildID: guild,
		tx:      tx,
		census:  p.census,
	}, nil
}

type pgTrialAPITx struct {
	guildID string
	tx      pgx.Tx
	census  *telemetry.Census
}

func (p *pgTrialAPITx) Commit(ctx context.Context) error {
	_, span := p.census.StartSpan(ctx, "pgTrialAPITx.Commit")
	defer span.End()

	return p.tx.Commit(ctx)
}

func (p *pgTrialAPITx) Rollback(ctx context.Context) error {
	_, span := p.census.StartSpan(ctx, "pgTrialAPITx.Rollback")
	defer span.End()

	err := p.tx.Rollback(ctx)
	if err != nil && err != pgx.ErrTxClosed {
		return err
	}
	return nil
}

func (p *pgTrialAPITx) GetTrial(ctx context.Context, name string) (Trial, error) {
	_, span := p.census.StartSpan(ctx, "pgTrialAPITx.GetTrial")
	defer span.End()

	r := p.tx.QueryRow(ctx, `
	SELECT event_data 
	FROM events 
	WHERE guild_id = $1 AND event_name = $2
	LIMIT 1`, p.guildID, name)

	var val []byte
	if err := r.Scan(&val); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrTrialNotExist
		}
		return nil, errors.Wrap(err, "could not retrieve event settings")
	}

	pTrial := ProtoTrial{}
	err := proto.Unmarshal(val, &pTrial)
	if err != nil {
		return nil, errors.Wrap(err, "trial record is corrupt")
	}

	return &protoTrial{
		protoTrial: &pTrial,
		census:     p.census,
	}, nil
}

func (p *pgTrialAPITx) AddTrial(ctx context.Context, name string) (Trial, error) {
	ctx, span := p.census.StartSpan(ctx, "pgTrialAPITx.AddTrial")
	defer span.End()

	trial, err := p.GetTrial(ctx, name)
	if err == ErrTrialNotExist {
		trial = &protoTrial{
			protoTrial: &ProtoTrial{Name: name},
			census:     p.census,
		}
		err = nil
	}
	return trial, err
}

func (p *pgTrialAPITx) SaveTrial(ctx context.Context, t Trial) error {
	ctx, span := p.census.StartSpan(ctx, "pgTrialAPITx.SaveTrial")
	defer span.End()

	serial, err := t.Serialize(ctx)
	if err != nil {
		return err
	}

	name := strings.ToLower(t.GetName(ctx))

	_, err = p.tx.Exec(ctx, `
	INSERT INTO events (guild_id, event_name, event_data) 
	VALUES ($1, $2, $3) 
	ON CONFLICT (guild_id, event_name) DO UPDATE
	SET event_data = EXCLUDED.event_data`, p.guildID, name, serial)

	return err
}

func (p *pgTrialAPITx) DeleteTrial(ctx context.Context, name string) error {
	ctx, span := p.census.StartSpan(ctx, "pgTrialAPITx.DeleteTrial")
	defer span.End()

	_, err := p.GetTrial(ctx, name)
	if err != nil {
		return err
	}

	name = strings.ToLower(name)

	_, err = p.tx.Exec(ctx, `
	DELETE FROM events 
	WHERE guild_id = $1 AND event_name = $2 
	LIMIT 1`, p.guildID, name)

	return err
}

func (p *pgTrialAPITx) GetTrials(ctx context.Context) []Trial {
	_, span := p.census.StartSpan(ctx, "pgTrialAPITx.GetTrials")
	defer span.End()

	t := make([]Trial, 0, 10)

	rs, err := p.tx.Query(ctx, `
	SELECT event_data 
	FROM events 
	WHERE guild_id = $1`, p.guildID)

	if err != nil && err != pgx.ErrNoRows {
		return nil
	}
	defer rs.Close()

	var val []byte
	for rs.Next() {
		val = val[:0] // truncate
		if err = rs.Scan(&val); err != nil {
			continue
		}

		pTrial := ProtoTrial{}
		err := proto.Unmarshal(val, &pTrial)
		if err == nil {
			t = append(t, &protoTrial{
				protoTrial: &pTrial,
				census:     p.census,
			})
		}
	}

	return t
}
