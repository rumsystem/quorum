package chain

import (
	"encoding/json"
	"errors"
	//"fmt"
	"math/rand"
	"sync"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/golang/glog"
	"github.com/oklog/ulid"

	blockstore "github.com/huo-ju/go-ipfs-blockstore"
	ds_sync "github.com/ipfs/go-datastore/sync"
	ds_badger "github.com/ipfs/go-ds-badger"
)

func (dbMgr *DbMgr) InitDb(datapath string) {
	badgerstorage, err := ds_badger.NewDatastore(datapath+"_bs", nil)
	bs := blockstore.NewBlockstore(ds_sync.MutexWrap(badgerstorage), "bs")

	dbMgr.GroupInfoDb, err = badger.Open(badger.DefaultOptions(datapath + "_groups"))
	if err != nil {
		glog.Fatal(err.Error())
	}

	dbMgr.TrxDb, err = badger.Open(badger.DefaultOptions(datapath + "_trx"))
	if err != nil {
		glog.Fatal(err.Error())
	}

	dbMgr.BlockDb, err = badger.Open(badger.DefaultOptions(datapath + "_block"))
	if err != nil {
		glog.Fatal(err.Error())
	}

	dbMgr.BlockStorage = bs
	dbMgr.DataPath = datapath

	glog.Infof("ChainCtx DbMgf initialized")
}

func (dbMgr *DbMgr) CloseDb() {
	dbMgr.GroupInfoDb.Close()
	dbMgr.TrxDb.Close()
	dbMgr.BlockDb.Close()
	glog.Infof("ChainCtx Db closed")
}

//save trx
func (dbMgr *DbMgr) AddTrx(trx Trx) error {
	err := dbMgr.TrxDb.Update(func(txn *badger.Txn) error {
		bytes, err := json.Marshal(trx)
		e := badger.NewEntry([]byte(trx.Msg.TrxId), bytes)
		err = txn.SetEntry(e)
		return err
	})

	if err != nil {
		return err
	}

	/*
		//test only, show db contents
		err = dbMgr.TrxDb.View(func(txn *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			opts.PrefetchSize = 10
			it := txn.NewIterator(opts)
			defer it.Close()
			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				k := item.Key()
				err := item.Value(func(v []byte) error {
					fmt.Printf("key=%s, value=%s\n", k, v)
					return nil
				})
				if err != nil {
					return err
				}
			}
			return nil
		})*/

	return nil
}

//rm Trx
func (dbMgr *DbMgr) RmTrx(trxId string) error {
	err := dbMgr.TrxDb.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(trxId))
		return err
	})

	return err
}

//update Trx
func (dbMgr *DbMgr) UpdTrxCons(trx Trx, consensusString string) error {
	return dbMgr.AddTrx(trx)
}

//get trx
func (dbMgr *DbMgr) GetTrx(trxId string) (Trx, error) {
	var trx Trx
	err := dbMgr.TrxDb.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(trxId))

		if err != nil {
			return err
		}

		trxBytes, err := item.ValueCopy(nil)

		if err != nil {
			return err
		}

		err = json.Unmarshal(trxBytes, &trx)

		if err != nil {
			return err
		}

		return nil
	})

	return trx, err
}

//Save Block
func (dbMgr *DbMgr) AddBlock(block Block) error {

	//AddBlock to blockDb
	err := dbMgr.BlockDb.Update(func(txn *badger.Txn) error {
		bytes, err := json.Marshal(block)
		e := badger.NewEntry([]byte(block.Cid), bytes)
		err = txn.SetEntry(e)
		return err
	})

	return err
}

//Rm Block
func (dbMgr *DbMgr) RmBlock(block Block) error {
	err := dbMgr.BlockDb.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(block.Cid))
		return err
	})

	return err
}

//Upd Block
func (dbMgr *DbMgr) UpdBlock(oldBlock, newBlock Block) error {
	err := dbMgr.AddBlock(newBlock)
	return err
}

//Get Block
func (dbMgr *DbMgr) GetBlock(blockId string) (Block, error) {
	var block Block
	err := dbMgr.BlockDb.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(blockId))
		if err != nil {
			return err
		}

		blockBytes, err := item.ValueCopy(nil)

		if err != nil {
			return err
		}

		err = json.Unmarshal(blockBytes, &block)
		return err
	})

	return block, err
}

//Get raw block ([]byte)
func (dbMgr *DbMgr) GetRawBlock(blockId string) ([]byte, error) {
	var raw []byte
	err := dbMgr.BlockDb.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(blockId))
		raw, err = item.ValueCopy(nil)
		return err
	})

	return raw, err
}

func (dbMgr *DbMgr) AddGroup(groupItem *GroupItem) error {

	//check if group exist
	err := dbMgr.GroupInfoDb.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(groupItem.GroupId))
		return err
	})

	if err == nil {
		return errors.New("Group with same GroupId existed")
	}

	//add group to db
	err = dbMgr.GroupInfoDb.Update(func(txn *badger.Txn) error {
		bytes, err := json.Marshal(groupItem)
		if err != nil {
			return err
		}
		e := badger.NewEntry([]byte(groupItem.GroupId), bytes)
		err = txn.SetEntry(e)
		return err
	})

	if err != nil {
		glog.Fatalf(err.Error())
	}

	/*
		//test only, show db contents
		err = dbMgr.GroupInfoDb.View(func(txn *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			opts.PrefetchSize = 10
			it := txn.NewIterator(opts)
			defer it.Close()
			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				k := item.Key()
				err := item.Value(func(v []byte) error {
					fmt.Printf("key=%s, value=%s\n", k, v)
					return nil
				})
				if err != nil {
					return err
				}
			}
			return nil
		})
		return err */

	return nil
}

func (dbMgr *DbMgr) UpdGroup(groupItem *GroupItem) error {
	//upd group to db
	err := dbMgr.GroupInfoDb.Update(func(txn *badger.Txn) error {
		bytes, err := json.Marshal(groupItem)
		if err != nil {
			return err
		}
		e := badger.NewEntry([]byte(groupItem.GroupId), bytes)
		err = txn.SetEntry(e)
		return err
	})

	return err
}

func (dbMgr *DbMgr) RmGroup(item *GroupItem) error {

	//check if group exist
	err := dbMgr.GroupInfoDb.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(item.GroupId))
		return err
	})

	if err != nil {
		return err
	}

	//delete group
	err = dbMgr.GroupInfoDb.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(item.GroupId))
		return err
	})

	if err != nil {
		return err
	}

	/*
		//test only, show db contents
		err = dbMgr.GroupInfoDb.View(func(txn *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			opts.PrefetchSize = 10
			it := txn.NewIterator(opts)
			defer it.Close()
			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				k := item.Key()
				err := item.Value(func(v []byte) error {
					fmt.Printf("key=%s, value=%s\n", k, v)
					return nil
				})
				if err != nil {
					return err
				}
			}
			return nil
		})

		return err
	*/

	return nil

}

//Get group list
func (dbMgr *DbMgr) GetGroupsBytes() ([][]byte, error) {
	var groupItemList [][]byte
	//test only, show db contents
	err := dbMgr.GroupInfoDb.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {

			item := it.Item()
			err := item.Value(func(v []byte) error {

				bytes, err := item.ValueCopy(nil)
				if err != nil {
					return err
				}
				groupItemList = append(groupItemList, bytes)

				return nil
			})

			if err != nil {
				return err
			}
		}
		return nil
	})

	return groupItemList, err
}

//Get group list string
func (dbMgr *DbMgr) GetGroupsString() ([]string, error) {
	var groupItemList []string
	//test only, show db contents
	err := dbMgr.GroupInfoDb.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {

			item := it.Item()
			err := item.Value(func(v []byte) error {

				bytes, err := item.ValueCopy(nil)
				if err != nil {
					return err
				}
				groupItemList = append(groupItemList, string(bytes))

				return nil
			})

			if err != nil {
				return err
			}
		}
		return nil
	})

	return groupItemList, err
}

//for generate sequence ULID
type MonotonicULIDSource struct {
	sync.Mutex
	entropy  *rand.Rand
	lastMs   uint64
	lastULID ulid.ULID
}

func GenerateULID() ulid.ULID {
	entropy := rand.New(rand.NewSource(time.Unix(1000000, 0).UnixNano()))
	ulidSource := NewMonotonicULIDSource(entropy)
	now := time.Now()
	id, _ := ulidSource.New(now)

	return id
}

func NewMonotonicULIDSource(entropy *rand.Rand) *MonotonicULIDSource {
	initial, err := ulid.New(ulid.Now(), entropy)
	if err != nil {
		panic(err)
	}

	return &MonotonicULIDSource{
		entropy:  entropy,
		lastMs:   0,
		lastULID: initial,
	}
}

func (u *MonotonicULIDSource) New(t time.Time) (ulid.ULID, error) {
	u.Lock()
	defer u.Unlock()

	ms := ulid.Timestamp(t)
	var err error
	if ms > u.lastMs {
		u.lastMs = ms
		u.lastULID, err = ulid.New(ms, u.entropy)
		return u.lastULID, err
	}

	incrEntropy := incrementBytes(u.lastULID.Entropy())
	var dup ulid.ULID
	dup.SetTime(ms)

	if err := dup.SetEntropy(incrEntropy); err != nil {
		return dup, err
	}

	u.lastULID = dup
	u.lastMs = ms

	return dup, nil
}

func incrementBytes(in []byte) []byte {
	const (
		minByte byte = 0
		maxByte byte = 255
	)

	out := make([]byte, len(in))
	copy(out, in)

	leastSigByteIdx := len(out) - 1
	mostSigByteIdx := 0

	for i := leastSigByteIdx; i >= mostSigByteIdx; i-- {
		if out[i] == maxByte {
			out[i] = minByte
			continue
		}

		out[i]++
		return out
	}

	panic(ulid.ErrOverflow)
}
