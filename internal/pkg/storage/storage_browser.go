//go:build js && wasm
// +build js,wasm
package storage

import (
	"bytes"
	"context"
	"errors"
	"syscall/js"

	"github.com/hack-pad/go-indexeddb/idb"
)

type QSIndexDB struct {
	db   *idb.Database
	name string
	ctx  context.Context
}

type QSIndexDBEntry struct {
	Key   []byte `json:"key"`
	Value []byte `json:"value"`
}

func NewIndexDBEntry(key []byte, value []byte) QSIndexDBEntry {
	return QSIndexDBEntry{key, value}
}

var DefaultDBName = "quorum"

func (s *QSIndexDB) Init(store string) error {
	ctx := context.Background()
	openRequest, _ := idb.Global().Open(ctx, DefaultDBName, 1, func(db *idb.Database, oldVersion, newVersion uint) error {
		db.CreateObjectStore(store, idb.ObjectStoreOptions{})
		return nil
	})
	db, err := openRequest.Await(ctx)
	s.db = db
	s.name = store
	s.ctx = ctx

	return err
}

func (s *QSIndexDB) Close() error {
	return s.db.Close()
}

func (s *QSIndexDB) Set(key []byte, val []byte) error {
	txn, _ := s.db.Transaction(idb.TransactionReadWrite, s.name)
	store, _ := txn.ObjectStore(s.name)
	store.AddKey(BytesToArrayBuffer(key), BytesToArrayBuffer(val))
	return txn.Await(s.ctx)
}

func (s *QSIndexDB) Delete(key []byte) error {
	txn, _ := s.db.Transaction(idb.TransactionReadWrite, s.name)
	store, _ := txn.ObjectStore(s.name)
	store.Delete(BytesToArrayBuffer(key))
	return txn.Await(s.ctx)
}

func (s *QSIndexDB) Get(key []byte) ([]byte, error) {
	txn, _ := s.db.Transaction(idb.TransactionReadWrite, s.name)
	store, _ := txn.ObjectStore(s.name)
	req, err := store.Get(BytesToArrayBuffer(key))
	if err != nil {
		return nil, err
	}
	jsVal, err := req.Await(s.ctx)
	if err != nil {
		return nil, err
	}
	bytes := ArrayBufferToBytes(jsVal)
	if len(bytes) == 0 {
		return nil, errors.New("KeyNotFound")
	}
	return bytes, nil
}

func (s *QSIndexDB) PrefixForeach(prefix []byte, fn func([]byte, []byte, error) error) error {
	txn, _ := s.db.Transaction(idb.TransactionReadWrite, s.name)
	store, _ := txn.ObjectStore(s.name)
	kRange, err := idb.NewKeyRangeLowerBound(BytesToArrayBuffer(prefix), false)
	if err != nil {
		return err
	}
	cursorRequest, err := store.OpenCursorRange(kRange, idb.CursorNext)
	if err != nil {
		return err
	}
	return cursorRequest.Iter(s.ctx, func(cursor *idb.CursorWithValue) error {
		key, err := cursor.Cursor.Key()
		if err != nil {
			return err
		}
		value, err := cursor.Value()
		if err != nil {
			return err
		}
		ferr := fn(ArrayBufferToBytes(key), ArrayBufferToBytes(value), nil)
		if ferr != nil {
			return ferr
		}
		return nil
	})
}

// for reverse, prefix is the upper bound, and valid is the actual prefix
func (s *QSIndexDB) PrefixForeachKey(prefix []byte, valid []byte, reverse bool, fn func([]byte, error) error) error {
	txn, _ := s.db.Transaction(idb.TransactionReadWrite, s.name)
	store, _ := txn.ObjectStore(s.name)
	if !reverse {
		kRange, err := idb.NewKeyRangeLowerBound(BytesToArrayBuffer(prefix), false)
		if err != nil {
			return err
		}
		cursorRequest, err := store.OpenKeyCursorRange(kRange, idb.CursorNext)
		if err != nil {
			return err
		}
		return cursorRequest.Iter(s.ctx, func(cursor *idb.Cursor) error {
			key, err := cursor.Key()
			if err != nil {
				return err
			}
			ferr := fn(ArrayBufferToBytes(key), nil)
			if ferr != nil {
				return ferr
			}
			return nil
		})
	} else {
		kRange, err := idb.NewKeyRangeUpperBound(BytesToArrayBuffer(prefix), false)
		if err != nil {
			return err
		}
		cursorRequest, err := store.OpenKeyCursorRange(kRange, idb.CursorPrevious)
		if err != nil {
			return err
		}
		return cursorRequest.Iter(s.ctx, func(cursor *idb.Cursor) error {
			key, err := cursor.Key()
			if err != nil {
				return err
			}
			k := ArrayBufferToBytes(key)
			if bytes.HasPrefix(k, valid) {
				ferr := fn(k, nil)
				if ferr != nil {
					return ferr
				}
			}
			return nil
		})
	}
}

func (s *QSIndexDB) Foreach(fn func([]byte, []byte, error) error) error {
	txn, _ := s.db.Transaction(idb.TransactionReadWrite, s.name)
	store, _ := txn.ObjectStore(s.name)
	cursorRequest, err := store.OpenCursor(idb.CursorNext)
	if err != nil {
		return err
	}
	return cursorRequest.Iter(s.ctx, func(cursor *idb.CursorWithValue) error {
		key, err := cursor.Cursor.Key()
		if err != nil {
			return err
		}
		value, err := cursor.Value()
		if err != nil {
			return err
		}
		ferr := fn(ArrayBufferToBytes(key), ArrayBufferToBytes(value), nil)
		if ferr != nil {
			return ferr
		}
		return nil
	})
}

func (s *QSIndexDB) IsExist(key []byte) (bool, error) {
	txn, _ := s.db.Transaction(idb.TransactionReadWrite, s.name)
	store, _ := txn.ObjectStore(s.name)
	req, err := store.Get(BytesToArrayBuffer(key))
	if err != nil {
		return false, err
	}
	jsVal, err := req.Await(s.ctx)
	if err != nil {
		return false, err
	}
	bytes := ArrayBufferToBytes(jsVal)
	if len(bytes) == 0 {
		return false, nil
	}
	return true, nil
}

// For appdb, atomic batch write
func (s *QSIndexDB) BatchWrite(keys [][]byte, values [][]byte) error {
	txn, _ := s.db.Transaction(idb.TransactionReadWrite, s.name)
	store, _ := txn.ObjectStore(s.name)
	for i, k := range keys {
		v := values[i]
		store.AddKey(BytesToArrayBuffer(k), BytesToArrayBuffer(v))
	}
	return txn.Await(s.ctx)
}

// TODO: implement Sequence
func (s *QSIndexDB) GetSequence([]byte, uint64) (Sequence, error) {
	return nil, nil
}

func ArrayBufferToBytes(buffer js.Value) []byte {
	view := js.Global().Get("Uint8Array").New(buffer)
	dataLen := view.Length()
	data := make([]byte, dataLen)
	if js.CopyBytesToGo(data, view) != dataLen {
		panic("expected to copy all bytes")
	}
	return data
}

func BytesToArrayBuffer(bytes []byte) js.Value {
	jsVal := js.Global().Get("Uint8Array").New(len(bytes))
	js.CopyBytesToJS(jsVal, bytes)
	return jsVal
}
