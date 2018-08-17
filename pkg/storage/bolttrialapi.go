package storage

import (
	"strings"

	bolt "github.com/coreos/bbolt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

// ErrTrialNotExist is the error returned if a trial does not exist
var ErrTrialNotExist = errors.New("trial does not exist")

type boltTrialAPI struct {
	db *bolt.DB
}

// NewBoltTrialAPI constructs a boltDB-backed TrialAPI
func NewBoltTrialAPI(db *bolt.DB) (TrialAPI, error) {
	b := boltTrialAPI{
		db: db,
	}

	return &b, nil
}

func (b *boltTrialAPI) NewTransaction(guild string, writable bool) (TrialAPITx, error) {
	bucketName := []byte(guild)

	err := b.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return errors.Wrap(err, "could not create bucket")
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	tx, err := b.db.Begin(writable)
	if err != nil {
		return nil, err
	}
	return &boltTrialAPITx{
		bucketName: bucketName,
		tx:         tx,
	}, nil
}

type boltTrialAPITx struct {
	bucketName []byte
	tx         *bolt.Tx
}

func (b *boltTrialAPITx) Commit() error {
	return b.tx.Commit()
}

func (b *boltTrialAPITx) Rollback() error {
	err := b.tx.Rollback()
	if err != nil && err != bolt.ErrTxClosed {
		return err
	}
	return nil
}

func (b *boltTrialAPITx) AddTrial(name string) (Trial, error) {
	name = strings.ToLower(name)

	user, err := b.GetTrial(name)
	if err == ErrTrialNotExist {
		user = &boltTrial{
			protoTrial: &ProtoTrial{Name: name},
		}
		err = nil
	}
	return user, err
}

func (b *boltTrialAPITx) SaveTrial(t Trial) error {
	bucket := b.tx.Bucket(b.bucketName)

	serial, err := t.Serialize()
	if err != nil {
		return err
	}

	return bucket.Put([]byte(strings.ToLower(t.GetName())), serial)
}

func (b *boltTrialAPITx) GetTrial(name string) (Trial, error) {
	bucket := b.tx.Bucket(b.bucketName)

	val := bucket.Get([]byte(strings.ToLower(name)))
	if val == nil {
		val = bucket.Get([]byte(name))
		if val == nil {
			return nil, ErrTrialNotExist
		}
	}

	protoTrial := ProtoTrial{}
	err := proto.Unmarshal(val, &protoTrial)
	if err != nil {
		return nil, errors.Wrap(err, "trial record is corrupt")
	}

	return &boltTrial{&protoTrial}, nil
}

func (b *boltTrialAPITx) DeleteTrial(name string) error {
	_, err := b.GetTrial(name)
	if err != nil {
		return err
	}

	name = strings.ToLower(name)
	bucket := b.tx.Bucket(b.bucketName)

	return bucket.Delete([]byte(name))
}

func (b *boltTrialAPITx) GetTrials() []Trial {
	bucket := b.tx.Bucket(b.bucketName)

	t := make([]Trial, 0, 10)
	_ = bucket.ForEach(func(k []byte, v []byte) error {
		protoTrial := ProtoTrial{}
		err := proto.Unmarshal(v, &protoTrial)
		if err == nil {
			t = append(t, &boltTrial{&protoTrial})
		}

		return nil
	})

	return t
}
