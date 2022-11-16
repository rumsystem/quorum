//go:build !js
// +build !js

package storage

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	bolt "go.etcd.io/bbolt"
)

const (
	boltAllocSize      = 8 * 1024 * 1024
	mmapSize           = 536870912 // Specifies the initial mmap size of bolt.
	sequenceBucketName = "__sequence__"
)

type Store struct {
	db           *bolt.DB
	databasePath string
	bucket       []byte
	ctx          context.Context
}

func getDBPath(dir string, bucket string) string {
	return filepath.Join(dir, bucket+".db")
}

func (s *Store) Init(path string) error {
	return nil
}

// OpenDB one bucket as a db
func OpenDB(dir, bucket string) (*bolt.DB, error) {
	if err := utils.EnsureDir(dir); err != nil {
		dbmgr_log.Errorf("check or create directory failed: %w", err)
		return nil, err
	}

	dbPath := getDBPath(dir, bucket)
	db, err := bolt.Open(
		dbPath,
		0644,
		&bolt.Options{
			Timeout:         1 * time.Second,
			InitialMmapSize: mmapSize,
		},
	)
	if err != nil {
		if errors.Is(err, bolt.ErrTimeout) {
			return nil, errors.New("can not obtain database lock, database may be in use by another process")
		}
		return nil, err
	}
	db.AllocSize = boltAllocSize

	return db, nil
}

func NewStore(ctx context.Context, dir string, bucket string) (*Store, error) {
	if err := utils.EnsureDir(dir); err != nil {
		dbmgr_log.Errorf("check or create directory failed: %w", err)
		return nil, err
	}

	db, err := OpenDB(dir, bucket)
	if err != nil {
		return nil, err
	}

	if err := db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	store := Store{
		db:           db,
		bucket:       []byte(bucket),
		databasePath: getDBPath(dir, bucket),
		ctx:          ctx,
	}

	return &store, nil
}

func createBuckets(tx *bolt.Tx, buckets ...[]byte) error {
	for _, bucket := range buckets {
		if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
			return err
		}
	}
	return nil
}

// ClearDB removes the previously stored database in the data directory.
func (s *Store) ClearDB() error {
	if err := os.Remove(s.databasePath); err != nil {
		return errors.New(fmt.Sprintf("could not remove database file: %s", err))
	}
	return nil
}

// Close closes the underlying BoltDB database.
func (s *Store) Close() error {
	return s.db.Close()
}

// DatabasePath at which this database writes files.
func (s *Store) DatabasePath() string {
	return s.databasePath
}

func (s *Store) Set(key []byte, val []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(s.bucket)
		return bucket.Put(key, val)
	})
}

func (s *Store) Delete(key []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(s.bucket)
		return bucket.Delete(key)
	})
}

// Get retrieves the value for a key in the bucket. Returns a nil value if the key does not exist or if the key is a nested bucket.
func (s *Store) Get(key []byte) ([]byte, error) {
	if key == nil || len(key) == 0 {
		return nil, rumerrors.ErrEmptyKey
	}

	var val []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(s.bucket)
		val = bucket.Get(key)
		if val != nil {
			val = val[:]
		}
		return nil
	})
	if err != nil {
		dbmgr_log.Warnf("kvdb Get %s failed: %s", key, err)
		return nil, err
	}

	return val, nil
}

func (s *Store) IsExist(key []byte) (bool, error) {
	val, err := s.Get(key)
	return val != nil, err
}

func (s *Store) PrefixDelete(prefix []byte) (int, error) {
	dbmgr_log.Debugf("delete key by prefix: %s", prefix)

	matched := 0

	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(s.bucket)
		c := bucket.Cursor()
		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			dbmgr_log.Debugf("delete key %s", k)
			if err := c.Delete(); err != nil {
				return err
			}
			matched += 1
		}
		return nil
	})

	return matched, err
}

func (s *Store) PrefixCondDelete(prefix []byte, fn func(k []byte, v []byte, err error) (bool, error)) (int, error) {
	matched := 0

	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(s.bucket)
		c := bucket.Cursor()

		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			ok, err := fn(k, v, nil)
			if err != nil {
				return err
			}
			if ok {
				if err := c.Delete(); err != nil {
					return err
				}
				matched += 1
			}
		}

		return nil
	})

	return matched, err
}

func (s *Store) PrefixForeachKey(prefix []byte, valid []byte, reverse bool, fn func([]byte, error) error) (int, error) {
	matched := 0

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(s.bucket)
		c := bucket.Cursor()
		if reverse {
			for k, _ := c.Last(); k != nil; k, _ = c.Prev() {
				if !bytes.HasPrefix(k, valid) {
					continue
				}
				if err := fn(k, nil); err != nil {
					return err
				}
				matched += 1
			}
		} else {
			for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, valid); k, _ = c.Next() {
				if err := fn(k, nil); err != nil {
					return err
				}
				matched += 1
			}
		}

		return nil
	})

	return matched, err
}

func (s *Store) PrefixForeach(prefix []byte, fn func([]byte, []byte, error) error) error {
	return s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(s.bucket)
		if bucket == nil {
			panic("bucket is nil")
		}
		c := bucket.Cursor()
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			if err := fn(k, v, nil); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Store) Foreach(fn func(k []byte, v []byte, err error) error) error {
	return s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(s.bucket)
		return bucket.ForEach(func(k, v []byte) error {
			return fn(k, v, nil)
		})
	})
}

func (s *Store) BatchWrite(keys [][]byte, vals [][]byte) error {
	if len(keys) != len(vals) {
		return errors.New("keys' and values' length should be equal")
	}

	tx, err := s.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	bucket := tx.Bucket(s.bucket)
	for i, k := range keys {
		if err := bucket.Put(k, vals[i]); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (s *Store) GetSequence(key []byte, bandwidth uint64) (Sequence, error) {
	if len(key) == 0 {
		return nil, errors.New("empty key")
	}
	if bandwidth == 0 {
		return nil, errors.New("zero bandwidth")
	}

	seq := &StoreSequence{
		store:     s,
		key:       key,
		next:      0,
		leased:    0,
		bandwidth: bandwidth,
	}
	err := seq.updateLease()
	return seq, err
}

func CreateDb(path string) (*DbMgr, error) {
	ctx := context.Background()
	groupDb, err := NewStore(ctx, path, "groups")
	if err != nil {
		return nil, err
	}
	dataDb, err := NewStore(ctx, path, "db")
	if err != nil {
		return nil, err
	}

	manager := DbMgr{
		GroupInfoDb: groupDb,
		Db:          dataDb,
		Auth:        nil,
		DataPath:    path,
	}
	return &manager, nil
}

// implement Sequence
type (
	StoreSequence struct {
		lock      sync.Mutex
		store     *Store
		key       []byte
		next      uint64
		leased    uint64
		bandwidth uint64
	}
)

var sequenceStore *Store = nil

func InitSequenceDB(dir string) error {
	if sequenceStore == nil {
		ctx := context.Background()
		var err error
		sequenceStore, err = NewStore(ctx, dir, sequenceBucketName)
		if err != nil {
			return err
		}
	}

	return nil
}

// Next would return the next integer in the sequence, updating the lease by running a transaction
// if needed.
func (seq *StoreSequence) Next() (uint64, error) {
	seq.lock.Lock()
	defer seq.lock.Unlock()
	if seq.next >= seq.leased {
		if err := seq.updateLease(); err != nil {
			return 0, err
		}
	}
	val := seq.next
	seq.next++
	return val, nil
}

// Release the leased sequence to avoid wasted integers. This should be done right
// before closing the associated DB. However it is valid to use the sequence after
// it was released, causing a new lease with full bandwidth.
func (seq *StoreSequence) Release() error {
	seq.lock.Lock()
	defer seq.lock.Unlock()
	err := seq.store.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(seq.store.bucket)
		val := bucket.Get(seq.key)

		var num uint64
		num = binary.BigEndian.Uint64(val)

		if num == seq.leased {
			var buf [8]byte
			binary.BigEndian.PutUint64(buf[:], seq.next)
			return bucket.Put(seq.key, buf[:])
		}

		return nil
	})
	if err != nil {
		return err
	}
	seq.leased = seq.next
	return nil
}

func (seq *StoreSequence) updateLease() error {
	return seq.store.db.Update(func(tx *bolt.Tx) error {
		val, err := seq.store.Get(seq.key)
		if err != nil {
			return err
		}
		if val == nil {
			seq.next = 0
		} else {
			var num uint64
			num = binary.BigEndian.Uint64(val)
			seq.next = num
		}

		lease := seq.next + seq.bandwidth
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], lease)
		bucket := tx.Bucket(seq.store.bucket)
		if err := bucket.Put(seq.key, buf[:]); err != nil {
			return err
		}
		seq.leased = lease
		return nil
	})
}
