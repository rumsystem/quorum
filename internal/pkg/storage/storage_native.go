//go:build !js
// +build !js

package storage

import (
	"errors"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/dgraph-io/badger/v3/options"
)

type QSBadger struct {
	db *badger.DB
}

var DefaultLogFileSize int64 = 16 << 20
var DefaultMemTableSize int64 = 8 << 20
var DefaultMaxEntries uint32 = 50000
var DefaultBlockCacheSize int64 = 32 << 20
var DefaultCompressionType = options.Snappy
var DefaultPrefetchSize = 10

func (s *QSBadger) Init(path string) error {
	var err error
	s.db, err = badger.Open(badger.DefaultOptions(path).WithValueLogFileSize(DefaultLogFileSize).WithMemTableSize(DefaultMemTableSize).WithValueLogMaxEntries(DefaultMaxEntries).WithBlockCacheSize(DefaultBlockCacheSize).WithCompression(DefaultCompressionType).WithLoggingLevel(badger.ERROR))
	if err != nil {
		return err
	}
	return nil
}

func (s *QSBadger) Close() error {
	return s.db.Close()
}

func (s *QSBadger) Set(key []byte, val []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry(key, val)
		err := txn.SetEntry(e)
		return err
	})
}

func (s *QSBadger) Delete(key []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(key))
		return err
	})

}

func (s *QSBadger) Get(key []byte) ([]byte, error) {
	var val []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		val, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}
		return nil
	})
	return val, err
}

func (s *QSBadger) IsExist(key []byte) (bool, error) {
	var ret bool

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 1
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		it.Seek(key)
		ret = it.ValidForPrefix(key)
		return nil
	})

	if err == nil {
		return ret, nil
	}
	return false, err
}

func (s *QSBadger) PrefixDelete(prefix []byte) error {
	return s.PrefixForeachKey(prefix, prefix, false, func(k []byte, err error) error {
		if err != nil {
			return err
		}
		return s.Delete(k)
	})
}

func (s *QSBadger) PrefixCondDelete(prefix []byte, fn func(k []byte, v []byte, err error) (bool, error)) error {
	return s.PrefixForeach(prefix, func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		del, err := fn(k, v, err)
		if err != nil {
			return err
		}
		if del {
			return s.Delete(k)
		}
		return nil
	})
}

func (s *QSBadger) PrefixForeach(prefix []byte, fn func([]byte, []byte, error) error) error {
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = DefaultPrefetchSize
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			key := item.KeyCopy(nil)
			val, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			ferr := fn(key, val, nil)
			if ferr != nil {
				return ferr
			}
		}
		return nil
	})
	return err
}

func (s *QSBadger) PrefixForeachKey(prefix []byte, valid []byte, reverse bool, fn func([]byte, error) error) error {
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 20
		opts.PrefetchValues = false
		opts.Reverse = reverse
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(valid); it.Next() {
			item := it.Item()
			key := item.KeyCopy(nil)
			ferr := fn(key, nil)
			if ferr != nil {
				return ferr
			}
		}
		return nil
	})
	return err
}

func (s *QSBadger) Foreach(fn func([]byte, []byte, error) error) error {
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = DefaultPrefetchSize
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.KeyCopy(nil)
			val, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			ferr := fn(key, val, nil)
			if ferr != nil {
				return ferr
			}
		}
		return nil
	})
	return err
}

func (s *QSBadger) BatchWrite(keys [][]byte, values [][]byte) error {
	if len(keys) != len(values) {
		return errors.New("keys' and values' length should be equal")
	}

	txn := s.db.NewTransaction(true)
	defer txn.Discard()

	for i, k := range keys {
		v := values[i]
		e := badger.NewEntry(k, v)
		err := txn.SetEntry(e)
		if err != nil {
			return err
		}
	}
	return txn.Commit()

}

func (s *QSBadger) GetSequence(key []byte, bandwidth uint64) (Sequence, error) {
	return s.db.GetSequence(key, bandwidth)
}

func CreateDb(path string) (*DbMgr, error) {
	var err error
	groupDb := QSBadger{}
	dataDb := QSBadger{}
	err = groupDb.Init(path + "_groups")
	if err != nil {
		return nil, err
	}

	err = dataDb.Init(path + "_db")
	if err != nil {
		return nil, err
	}

	manager := DbMgr{&groupDb, &dataDb, nil, path}
	return &manager, nil
}
