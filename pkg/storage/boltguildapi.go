package storage

import (
	"bytes"

	bolt "github.com/coreos/bbolt"
	"github.com/golang/protobuf/proto"
	"github.com/gsmcwhirter/go-util/v3/errors"
)

// ErrGuildNotExist is the error returned if a guild does not exist
var ErrGuildNotExist = errors.New("guild does not exist")

var settingsBucket = []byte("GuildRecords")

type boltGuildAPI struct {
	db         *bolt.DB
	bucketName []byte
}

// NewBoltGuildAPI constructs a boltDB-backed GuildAPI
func NewBoltGuildAPI(db *bolt.DB) (GuildAPI, error) {
	b := boltGuildAPI{
		db:         db,
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

func (b *boltGuildAPI) AllGuilds() ([]string, error) {
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

func (b *boltGuildAPI) NewTransaction(writable bool) (GuildAPITx, error) {
	tx, err := b.db.Begin(writable)
	if err != nil {
		return nil, err
	}
	return &boltGuildAPITx{
		bucketName: b.bucketName,
		tx:         tx,
	}, nil
}

type boltGuildAPITx struct {
	bucketName []byte
	tx         *bolt.Tx
}

func (b *boltGuildAPITx) Commit() error {
	return b.tx.Commit()
}

func (b *boltGuildAPITx) Rollback() error {
	err := b.tx.Rollback()
	if err != nil && err != bolt.ErrTxClosed {
		return err
	}
	return nil
}

func (b *boltGuildAPITx) AddGuild(name string) (Guild, error) {
	guild, err := b.GetGuild(name)
	if err == ErrGuildNotExist {
		guild = &boltGuild{
			protoGuild: &ProtoGuild{Name: name},
		}
		err = nil
	}
	return guild, err
}

func (b *boltGuildAPITx) SaveGuild(guild Guild) error {
	bucket := b.tx.Bucket(b.bucketName)

	serial, err := guild.Serialize()
	if err != nil {
		return err
	}

	return bucket.Put([]byte(guild.GetName()), serial)
}

func (b *boltGuildAPITx) GetGuild(name string) (Guild, error) {
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

	return &boltGuild{&protoGuild}, nil
}
