package chain

import (
	"encoding/json"
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
func (dbMgr *DbMgr) AddTrx(trx quorumpb.Trx) error {
	key := TRX_PREFIX + trx.Msg.TrxId
	err := dbMgr.Db.Update(func(txn *badger.Txn) error {
		bytes, err := proto.Marshal(&trx)
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
func (dbMgr *DbMgr) UpdTrxCons(trx quorumpb.Trx, consensusString string) error {
	return dbMgr.AddTrx(trx)
}

//get trx
func (dbMgr *DbMgr) GetTrx(trxId string) (quorumpb.Trx, error) {
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

	return trx, err
}

//Save Block
func (dbMgr *DbMgr) AddBlock(block quorumpb.Block) error {

	key := BLK_PREFIX + block.Cid
	//AddBlock to blockDb
	err := dbMgr.Db.Update(func(txn *badger.Txn) error {
		bytes, err := proto.Marshal(&block)
		e := badger.NewEntry([]byte(key), bytes)
		err = txn.SetEntry(e)
		return err
	})

	return err
}

//Rm Block
func (dbMgr *DbMgr) RmBlock(block quorumpb.Block) error {
	key := BLK_PREFIX + block.Cid
	err := dbMgr.Db.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(key))
		return err
	})

	return err
}

//Upd Block
func (dbMgr *DbMgr) UpdBlock(oldBlock, newBlock quorumpb.Block) error {
	err := dbMgr.AddBlock(newBlock)
	return err
}

//Get Block
func (dbMgr *DbMgr) GetBlock(blockId string) (quorumpb.Block, error) {
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

	return block, err
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

func (dbMgr *DbMgr) UpdBlkSeq(block quorumpb.Block) error {
	key := GRP_PREFIX + BLK_PREFIX + SEQ_PREFIX + block.GroupId + "_" + fmt.Sprint(block.BlockNum)

	//update BlockSeqDb
	err := dbMgr.Db.Update(func(txn *badger.Txn) error {
		//b := make([]byte, 8)
		//binary.LittleEndian.PutUint64(b, uint64(block.BlockNum))
		e := badger.NewEntry([]byte(key), []byte(block.Cid))
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

func (dbMgr *DbMgr) AddGrpCtnt(block quorumpb.Block) error {
	for _, trx := range block.Trxs {

		var ctnItem *GroupContentItem
		ctnItem = &GroupContentItem{}

		ctnItem.TrxId = trx.Msg.TrxId
		ctnItem.Publisher = trx.Msg.Sender
		ctnItem.Content = trx.Data
		ctnItem.TimeStamp = trx.Msg.TimeStamp
		ctnBytes, err := json.Marshal(ctnItem)
		if err != nil {
			return err
		}

		key := GRP_PREFIX + CNT_PREFIX + block.GroupId + "_" + trx.Msg.TrxId + "_" + fmt.Sprint(trx.Msg.TimeStamp)

		glog.Infof("Add trx with key %s", key)
		//update ContentDb
		err = dbMgr.Db.Update(func(txn *badger.Txn) error {
			e := badger.NewEntry([]byte(key), ctnBytes)
			err := txn.SetEntry(e)
			return err
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func (dbMgr *DbMgr) GetGrpCtnt(groupId string) ([]*GroupContentItem, error) {
	var ctnList []*GroupContentItem
	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		key := GRP_PREFIX + CNT_PREFIX + groupId + "_"
		glog.Infof("Get Key Prefix %s", key)
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek([]byte(key)); it.ValidForPrefix([]byte(key)); it.Next() {
			glog.Infof("Append")
			item := it.Item()
			err := item.Value(func(v []byte) error {
				var contentitem *GroupContentItem
				ctnerr := json.Unmarshal(v, &contentitem)
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
