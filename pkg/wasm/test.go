//go:build js && wasm
// +build js,wasm

package wasm

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/google/orderedcode"

	quorumStorage "github.com/rumsystem/quorum/internal/pkg/storage"
)

func IndexDBTest() {
	dbMgr := quorumStorage.QSIndexDB{}
	err := dbMgr.Init("test")
	if err != nil {
		panic(err)
	}

	{
		k := []byte("key")
		v := []byte("value")
		err = dbMgr.Set(k, v)
		if err != nil {
			panic(err)
		}

		val, err := dbMgr.Get(k)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, val) {
			panic("Get")
		}

		err = dbMgr.Delete(k)
		if err != nil {
			panic(err)
		}

		val, err = dbMgr.Get(k)
		if err == nil {
			panic("key should not found")
		}
		exist, err := dbMgr.IsExist(k)
		if !(exist == false && err == nil) {
			panic("dbMgr.IsExist")
		}
	}

	{
		keys := [][]byte{}
		values := [][]byte{}
		keyPrefix := "key"
		i := 0
		for i < 100 {
			k, _ := orderedcode.Append(nil, keyPrefix, "-", orderedcode.Infinity, uint64(i))
			keys = append(keys, k)
			values = append(values, []byte(fmt.Sprintf("value-%d", i)))
			i += 1
		}
		err := dbMgr.BatchWrite(keys, values)
		if err != nil {
			panic(err)
		}

		rKey, _ := orderedcode.Append(nil, keyPrefix, "-", orderedcode.Infinity, uint64(100))
		i = 0
		err = dbMgr.PrefixForeachKey(rKey, []byte(keyPrefix), true, func(k []byte, err error) error {
			i += 1
			curKey, _ := orderedcode.Append(nil, keyPrefix, "-", orderedcode.Infinity, uint64(100-i))
			if !bytes.Equal(k, curKey) {
				return errors.New("wrong key")
			}
			return nil
		})
		if err != nil {
			panic(err)
		}

		i = 0
		err = dbMgr.PrefixForeachKey([]byte(keyPrefix), []byte(keyPrefix), false, func(k []byte, err error) error {
			curKey, _ := orderedcode.Append(nil, keyPrefix, "-", orderedcode.Infinity, uint64(i))
			i += 1
			if !bytes.Equal(k, curKey) {
				return errors.New("wrong key")
			}
			return nil
		})
		if err != nil {
			panic(err)
		}

		dbMgr.Foreach(func(k []byte, v []byte, err error) error {
			println(string(k), string(v))
			return nil
		})

		for _, k := range keys {
			err = dbMgr.Delete([]byte(k))
			if err != nil {
				panic(err)
			}
		}

		println("Test Done: OK")
	}

	// {
	// 	// this won't pass,
	// 	// cursors can not be nested in indexeddb
	// 	dbMgr2 := quorumStorage.QSIndexDB{}
	// 	err = dbMgr2.Init("test2")
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	// nest
	// 	keys := [][]byte{}
	// 	values := [][]byte{}
	// 	keyPrefix := "key"
	// 	i := 0
	// 	for i < 10 {
	// 		k, _ := orderedcode.Append(nil, keyPrefix, "-", orderedcode.Infinity, uint64(i))
	// 		keys = append(keys, k)
	// 		values = append(values, []byte(fmt.Sprintf("value-%d", i)))
	// 		i += 1
	// 	}
	// 	err = dbMgr.BatchWrite(keys, values)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	err = dbMgr2.BatchWrite(keys, values)
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	err = dbMgr.PrefixForeach([]byte(keyPrefix), func(k []byte, v []byte, err error) error {
	// 		println("dbMgr: ", string(k), string(v))
	// 		err = dbMgr2.PrefixForeach([]byte(keyPrefix), func(k []byte, v []byte, err error) error {
	// 			println("dbMgr2: ", string(k), string(v))
	// 			return nil
	// 		})
	// 		return nil
	// 	})

	// 	for _, k := range keys {
	// 		err = dbMgr.Delete([]byte(k))
	// 		if err != nil {
	// 			panic(err)
	// 		}
	// 		err = dbMgr2.Delete([]byte(k))
	// 		if err != nil {
	// 			panic(err)
	// 		}
	// 	}
	// }
}
