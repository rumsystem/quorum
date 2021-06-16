package chain

import (
	"errors"
	"fmt"

	"math/rand"
	"sync"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/golang/glog"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"github.com/oklog/ulid"
	"google.golang.org/protobuf/proto"
)

const TRX_PREFIX string = "trx_"
const BLK_PREFIX string = "blk_"
const SEQ_PREFIX string = "seq_"
const GRP_PREFIX string = "grp_"
const CNT_PREFIX string = "cnt_"
const ATH_PREFIX string = "ath_"

func (dbMgr *DbMgr) InitDb(datapath string, dbopts *DbOption) {
	var err error
	dbMgr.GroupInfoDb, err = badger.Open(badger.DefaultOptions(datapath + "_groups").WithValueLogFileSize(dbopts.LogFileSize).WithMemTableSize(dbopts.MemTableSize).WithValueLogMaxEntries(dbopts.LogMaxEntries).WithBlockCacheSize(dbopts.BlockCacheSize).WithCompression(dbopts.Compression))
	if err != nil {
		glog.Fatal(err.Error())
	}

	dbMgr.Db, err = badger.Open(badger.DefaultOptions(datapath + "_db").WithValueLogFileSize(dbopts.LogFileSize).WithMemTableSize(dbopts.MemTableSize).WithValueLogMaxEntries(dbopts.LogMaxEntries).WithBlockCacheSize(dbopts.BlockCacheSize).WithCompression(dbopts.Compression))
	if err != nil {
		glog.Fatal(err.Error())
	}

	dbMgr.DataPath = datapath

	glog.Infof("ChainCtx DbMgf initialized")
}

func (dbMgr *DbMgr) CloseDb() {
	dbMgr.GroupInfoDb.Close()
	dbMgr.Db.Close()
	glog.Infof("ChainCtx Db closed")
}

//save trx
func (dbMgr *DbMgr) AddTrx(trx *quorumpb.Trx) error {
	key := TRX_PREFIX + trx.TrxId
	err := dbMgr.Db.Update(func(txn *badger.Txn) error {
		bytes, err := proto.Marshal(trx)
		e := badger.NewEntry([]byte(key), bytes)
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
	key := TRX_PREFIX + trxId
	err := dbMgr.Db.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(key))
		return err
	})

	return err
}

//update Trx
func (dbMgr *DbMgr) UpdTrxCons(trx *quorumpb.Trx, consensusString string) error {
	return dbMgr.AddTrx(trx)
}

//get trx
func (dbMgr *DbMgr) GetTrx(trxId string) (*quorumpb.Trx, error) {
	var trx quorumpb.Trx
	key := TRX_PREFIX + trxId
	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))

		if err != nil {
			return err
		}

		trxBytes, err := item.ValueCopy(nil)

		if err != nil {
			return err
		}

		err = proto.Unmarshal(trxBytes, &trx)

		if err != nil {
			return err
		}

		return nil
	})

	return &trx, err
}

//Save Block
func (dbMgr *DbMgr) AddBlock(block *quorumpb.Block) error {

	key := BLK_PREFIX + block.BlockId
	//AddBlock to blockDb
	err := dbMgr.Db.Update(func(txn *badger.Txn) error {
		bytes, err := proto.Marshal(block)
		e := badger.NewEntry([]byte(key), bytes)
		err = txn.SetEntry(e)
		return err
	})

	return err
}

//Rm Block
func (dbMgr *DbMgr) RmBlock(blockCid string) error {
	key := BLK_PREFIX + blockCid
	err := dbMgr.Db.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(key))
		return err
	})

	return err
}

//Upd Block
func (dbMgr *DbMgr) UpdBlock(oldBlock, newBlock *quorumpb.Block) error {
	err := dbMgr.AddBlock(newBlock)
	return err
}

//Get Block
func (dbMgr *DbMgr) GetBlock(blockId string) (*quorumpb.Block, error) {
	var block quorumpb.Block
	key := BLK_PREFIX + blockId

	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		blockBytes, err := item.ValueCopy(nil)

		if err != nil {
			return err
		}

		err = proto.Unmarshal(blockBytes, &block)
		return err
	})

	return &block, err
}

//Get raw block ([]byte)
func (dbMgr *DbMgr) GetRawBlock(blockId string) ([]byte, error) {
	var raw []byte
	key := BLK_PREFIX + blockId
	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))

		if err != nil {
			return err
		}

		raw, err = item.ValueCopy(nil)
		return err
	})

	return raw, err
}

func (dbMgr *DbMgr) AddGroup(groupItem *quorumpb.GroupItem) error {
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
		bytes, err := proto.Marshal(groupItem)
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

	return nil
}

func (dbMgr *DbMgr) UpdGroup(groupItem *quorumpb.GroupItem) error {
	//upd group to db
	err := dbMgr.GroupInfoDb.Update(func(txn *badger.Txn) error {
		bytes, err := proto.Marshal(groupItem)
		if err != nil {
			return err
		}
		e := badger.NewEntry([]byte(groupItem.GroupId), bytes)
		err = txn.SetEntry(e)
		return err
	})

	return err
}

func (dbMgr *DbMgr) RmGroup(item *quorumpb.GroupItem) error {
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

func (dbMgr *DbMgr) UpdBlkSeq(block *quorumpb.Block) error {
	key := GRP_PREFIX + BLK_PREFIX + SEQ_PREFIX + block.GroupId + "_" + fmt.Sprint(block.BlockNum)
	err := dbMgr.Db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(key), []byte(block.BlockId))
		err := txn.SetEntry(e)
		return err
	})

	return err
}

func (dbMgr *DbMgr) GetBlkId(blockNum int64, groupId string) (string, error) {
	var blockId string
	key := GRP_PREFIX + BLK_PREFIX + SEQ_PREFIX + groupId + "_" + fmt.Sprint(blockNum)
	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))

		if err != nil {
			return err
		}

		blockIdBytes, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		blockId = string(blockIdBytes)
		return nil
	})

	return blockId, err
}

func (dbMgr *DbMgr) AddPost(trx *quorumpb.Trx) error {

	var ctnItem *quorumpb.PostItem
	ctnItem = &quorumpb.PostItem{}

	ctnItem.TrxId = trx.TrxId
	ctnItem.Publisher = trx.Sender
	ctnItem.Content = trx.Data
	ctnItem.TimeStamp = trx.TimeStamp
	ctnBytes, err := proto.Marshal(ctnItem)
	if err != nil {
		return err
	}

	key := GRP_PREFIX + CNT_PREFIX + trx.GroupId + "_" + trx.TrxId + "_" + fmt.Sprint(trx.TimeStamp)

	glog.Infof("Add trx with key %s", key)
	//update ContentDb
	err = dbMgr.Db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(key), ctnBytes)
		err := txn.SetEntry(e)
		return err
	})

	return err
}

func (dbMgr *DbMgr) GetGrpCtnt(groupId string) ([]*quorumpb.PostItem, error) {
	var ctnList []*quorumpb.PostItem
	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		key := GRP_PREFIX + CNT_PREFIX + groupId + "_"
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek([]byte(key)); it.ValidForPrefix([]byte(key)); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				contentitem := &quorumpb.PostItem{}
				ctnerr := proto.Unmarshal(v, contentitem)
				if ctnerr == nil {
					ctnList = append(ctnList, contentitem)
				}

				return ctnerr
			})

			if err != nil {
				return err
			}
		}

		return nil
	})

	return ctnList, err
}

func (dbMgr *DbMgr) UpdateBlkListItem(trx *quorumpb.Trx) (err error) {

	item := &quorumpb.BlockListItem{}

	if err := proto.Unmarshal(trx.Data, item); err != nil {
		return err
	}

	if item.Memo == "Add" {
		key := ATH_PREFIX + item.GroupId + "_" + item.UserId
		err = dbMgr.Db.Update(func(txn *badger.Txn) error {
			e := badger.NewEntry([]byte(key), trx.Data)
			err := txn.SetEntry(e)
			return err
		})
	} else if item.Memo == "Remove" {
		key := ATH_PREFIX + item.GroupId + "_" + item.UserId

		//check if group exist
		err = dbMgr.Db.View(func(txn *badger.Txn) error {
			_, err := txn.Get([]byte(key))
			return err
		})

		if err != nil {
			return err
		}

		err = dbMgr.Db.Update(func(txn *badger.Txn) error {
			err := txn.Delete([]byte(key))
			return err
		})

		return err
	} else {
		err := errors.New("unknow msgType")
		return err
	}

	return nil
}

func (dbMgr *DbMgr) GetBlkListItems() ([]*quorumpb.BlockListItem, error) {
	var blkList []*quorumpb.BlockListItem
	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		key := ATH_PREFIX
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek([]byte(key)); it.ValidForPrefix([]byte(key)); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				blkItem := &quorumpb.BlockListItem{}
				ctnerr := proto.Unmarshal(v, blkItem)
				if ctnerr == nil {
					blkList = append(blkList, blkItem)
				}

				return ctnerr
			})

			if err != nil {
				return err
			}
		}
		return nil
	})

	return blkList, err
}

func (dbMgr *DbMgr) IsBlocked(groupId, userId string) (bool, error) {
	key := ATH_PREFIX + groupId + "_" + userId

	//check if group exist
	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(key))
		return err
	})

	if err == nil {
		return true, nil
	}

	return false, err
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
