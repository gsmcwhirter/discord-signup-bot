package storage

import (
	"bytes"
	"context"

	bolt "github.com/coreos/bbolt"
	"github.com/golang/protobuf/proto"
	"github.com/gsmcwhirter/go-util/v4/census"
	"github.com/gsmcwhirter/go-util/v4/errors"
)

// ErrGuildNotExist is the error returned if a guild does not exist
var ErrGuildNotExist = errors.New("guild does not exist")

var settingsBucket = []byte("GuildRecords")

type boltGuildAPI struct {
	db         *bolt.DB
	census     *census.OpenCensus
	bucketName []byte
}

// NewBoltGuildAPI constructs a boltDB-backed GuildAPI
func NewBoltGuildAPI(ctx context.Context, db *bolt.DB, c *census.OpenCensus) (GuildAPI, error) {
	_, span := c.StartSpan(ctx, "boltGuildAPI.NewBoltGuildAPI")
	defer span.End()

	b := boltGuildAPI{
		db:         db,
		census:     c,
		bucketName: settingsBucket,
	}

	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(b.bucketName)
		if err != nil {
			return errors.Wrap(err, "could not create bucket")
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &b, nil
}

func (b *boltGuildAPI) AllGuilds(ctx context.Context) ([]string, error) {
	_, span := b.census.StartSpan(ctx, "boltGuildAPI.AllGuilds")
	defer span.End()

	var guilds []string

	err := b.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(bucketName []byte, b *bolt.Bucket) error {
			if !bytes.Equal(bucketName, settingsBucket) {
				guilds = append(guilds, string(bucketName))
			}

			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return guilds, nil
}

func (b *boltGuildAPI) NewTransaction(ctx context.Context, writable bool) (GuildAPITx, error) {
	_, span := b.census.StartSpan(ctx, "boltGuildAPI.NewTransaction")
	defer span.End()

	tx, err := b.db.Begin(writable)
	if err != nil {
		return nil, err
	}
	return &boltGuildAPITx{
		bucketName: b.bucketName,
		tx:         tx,
		census:     b.census,
	}, nil
}

type boltGuildAPITx struct {
	bucketName []byte
	tx         *bolt.Tx
	census     *census.OpenCensus
}

func (b *boltGuildAPITx) Commit(ctx context.Context) error {
	_, span := b.census.StartSpan(ctx, "boltGuildAPITx.Commit")
	defer span.End()

	return b.tx.Commit()
}

func (b *boltGuildAPITx) Rollback(ctx context.Context) error {
	_, span := b.census.StartSpan(ctx, "boltGuildAPITx.Rollback")
	defer span.End()

	err := b.tx.Rollback()
	if err != nil && err != bolt.ErrTxClosed {
		return err
	}
	return nil
}

func (b *boltGuildAPITx) AddGuild(ctx context.Context, name string) (Guild, error) {
	_, span := b.census.StartSpan(ctx, "boltGuildAPITx.AddGuild")
	defer span.End()

	guild, err := b.GetGuild(ctx, name)
	if err == ErrGuildNotExist {
		guild = &boltGuild{
			protoGuild: &ProtoGuild{Name: name},
		}
		err = nil
	}
	return guild, err
}

func (b *boltGuildAPITx) SaveGuild(ctx context.Context, guild Guild) error {
	_, span := b.census.StartSpan(ctx, "boltGuildAPITx.SaveGuild")
	defer span.End()

	bucket := b.tx.Bucket(b.bucketName)

	serial, err := guild.Serialize(ctx)
	if err != nil {
		return err
	}

	return bucket.Put([]byte(guild.GetName(ctx)), serial)
}

func (b *boltGuildAPITx) GetGuild(ctx context.Context, name string) (Guild, error) {
	_, span := b.census.StartSpan(ctx, "boltGuildAPITx.GetGuild")
	defer span.End()

	bucket := b.tx.Bucket(b.bucketName)

	val := bucket.Get([]byte(name))

	if val == nil {
		return nil, ErrGuildNotExist
	}

	protoGuild := ProtoGuild{}
	err := proto.Unmarshal(val, &protoGuild)
	if err != nil {
		return nil, errors.Wrap(err, "guild record is corrupt")
	}

	return &boltGuild{&protoGuild, b.census}, nil
}
