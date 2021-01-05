package chain

import (
	"errors"
	"fmt"

	badger "github.com/dgraph-io/badger/v3"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	logging "github.com/ipfs/go-log/v2"
	"google.golang.org/protobuf/proto"
)

var dbmgr_log = logging.Logger("dbmgr")

const TRX_PREFIX string = "trx" //trx
const BLK_PREFIX string = "blk" //block
const SEQ_PREFIX string = "seq" //sequence
const GRP_PREFIX string = "grp" //group
const CNT_PREFIX string = "cnt" //content
const ATH_PREFIX string = "ath" //auth
const PRD_PREFIX string = "prd" //producer
const ANN_PREFIX string = "ann" //announce
const SMA_PREFIX string = "sma" //schema
const CHD_PREFIX string = "chd" //cached

func (dbMgr *DbMgr) InitDb(datapath string, dbopts *DbOption) error {
	var err error
	dbMgr.GroupInfoDb, err = badger.Open(badger.DefaultOptions(datapath + "_groups").WithValueLogFileSize(dbopts.LogFileSize).WithMemTableSize(dbopts.MemTableSize).WithValueLogMaxEntries(dbopts.LogMaxEntries).WithBlockCacheSize(dbopts.BlockCacheSize).WithCompression(dbopts.Compression).WithLoggingLevel(badger.ERROR))
	if err != nil {
		return err
	}

	dbMgr.Db, err = badger.Open(badger.DefaultOptions(datapath + "_db").WithValueLogFileSize(dbopts.LogFileSize).WithMemTableSize(dbopts.MemTableSize).WithValueLogMaxEntries(dbopts.LogMaxEntries).WithBlockCacheSize(dbopts.BlockCacheSize).WithCompression(dbopts.Compression).WithLoggingLevel(badger.ERROR))
	if err != nil {
		return err
	}

	dbMgr.DataPath = datapath

	dbmgr_log.Infof("ChainCtx DbMgf initialized")
	return nil
}

func (dbMgr *DbMgr) CloseDb() {
	dbMgr.GroupInfoDb.Close()
	dbMgr.Db.Close()
	dbmgr_log.Infof("ChainCtx Db closed")
}

//save trx
func (dbMgr *DbMgr) AddTrx(trx *quorumpb.Trx, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + TRX_PREFIX + "_" + trx.TrxId
	return dbMgr.Db.Update(func(txn *badger.Txn) error {
		bytes, err := proto.Marshal(trx)
		e := badger.NewEntry([]byte(key), bytes)
		err = txn.SetEntry(e)
		return err
	})
}

//UNUSED
//rm Trx
func (dbMgr *DbMgr) RmTrx(trxId string, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + TRX_PREFIX + "_" + trxId
	return dbMgr.Db.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(key))
		return err
	})
}

//get trx
func (dbMgr *DbMgr) GetTrx(trxId string, prefix ...string) (*quorumpb.Trx, error) {
	nodeprefix := getPrefix(prefix...)
	var trx quorumpb.Trx
	key := nodeprefix + TRX_PREFIX + "_" + trxId
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

func (dbMgr *DbMgr) UpdTrx(trx *quorumpb.Trx, prefix ...string) error {
	return dbMgr.AddTrx(trx, prefix...)
}

func (dbMgr *DbMgr) IsTrxExist(trxId string, prefix ...string) (bool, error) {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + TRX_PREFIX + "_" + trxId

	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(key))
		return err
	})

	if err == nil {
		return true, nil
	}
	if err == badger.ErrKeyNotFound {
		return false, nil
	}
	return false, err
}

func (dbMgr *DbMgr) AddGensisBlock(gensisBlock *quorumpb.Block, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + BLK_PREFIX + "_" + gensisBlock.BlockId
	var chunk *quorumpb.BlockDbChunk
	chunk = &quorumpb.BlockDbChunk{}

	chunk.BlockId = gensisBlock.BlockId
	chunk.BlockItem = gensisBlock
	chunk.ParentBlockId = ""
	chunk.Height = 0

	return dbMgr.Db.Update(func(txn *badger.Txn) error {
		bytes, err := proto.Marshal(chunk)
		e := badger.NewEntry([]byte(key), bytes)
		err = txn.SetEntry(e)
		return err
	})
}

//check if block existed
func (dbMgr *DbMgr) IsBlockExist(blockId string, cached bool, prefix ...string) (bool, error) {
	nodeprefix := getPrefix(prefix...)
	var key string
	if cached {
		key = nodeprefix + CHD_PREFIX + "_" + BLK_PREFIX + "_" + blockId
	} else {
		key = nodeprefix + BLK_PREFIX + "_" + blockId
	}

	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(key))
		return err
	})

	if err == nil {
		return true, nil
	}
	if err == badger.ErrKeyNotFound {
		return false, nil
	}
	return false, err
}

//check if parent block existed
func (dbMgr *DbMgr) IsParentExist(parentBlockId string, cached bool, prefix ...string) (bool, error) {
	nodeprefix := getPrefix(prefix...)
	var pKey string
	if cached {
		pKey = nodeprefix + CHD_PREFIX + "_" + BLK_PREFIX + "_" + parentBlockId
	} else {
		pKey = nodeprefix + BLK_PREFIX + "_" + parentBlockId
	}

	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(pKey))
		return err
	})

	if err == badger.ErrKeyNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

//add block
func (dbMgr *DbMgr) AddBlock(newBlock *quorumpb.Block, cached bool, prefix ...string) error {
	//create new chunk
	var chunk *quorumpb.BlockDbChunk
	chunk = &quorumpb.BlockDbChunk{}
	chunk.BlockId = newBlock.BlockId
	chunk.BlockItem = newBlock

	if cached {
		chunk.Height = -1        //Set height of cached chunk to -1
		chunk.ParentBlockId = "" //Set parent of cached chund to empty ""
	} else {
		//try get parent chunk
		pChunk, err := dbMgr.getBlockChunk(newBlock.PrevBlockId, cached, prefix...)
		if err != nil {
			return err
		}

		//update parent chunk
		pChunk.SubBlockId = append(pChunk.SubBlockId, chunk.BlockId)
		err = dbMgr.saveBlockChunk(pChunk, cached, prefix...)
		if err != nil {
			return err
		}

		chunk.Height = pChunk.Height + 1     //increase height
		chunk.ParentBlockId = pChunk.BlockId //point to parent
	}

	//save chunk
	return dbMgr.saveBlockChunk(chunk, cached, prefix...)
}

//remove block
func (dbMgr *DbMgr) RmBlock(blockId string, cached bool, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
	var key string
	if cached {
		key = nodeprefix + CHD_PREFIX + "_" + BLK_PREFIX + "_" + blockId
	} else {
		key = nodeprefix + BLK_PREFIX + "_" + blockId
	}

	return dbMgr.Db.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(key))
		return err
	})
}

//get block by block_id
func (dbMgr *DbMgr) GetBlock(blockId string, cached bool, prefix ...string) (*quorumpb.Block, error) {
	pChunk, err := dbMgr.getBlockChunk(blockId, cached, prefix...)
	if err != nil {
		return nil, err
	}
	return pChunk.BlockItem, nil
}

func (dbMgr *DbMgr) GatherBlocksFromCache(newBlock *quorumpb.Block, cached bool, prefix ...string) ([]*quorumpb.Block, error) {
	nodeprefix := getPrefix(prefix...)
	var blocks []*quorumpb.Block
	blocks = append(blocks, newBlock)
	pointer1 := 0 //point to head
	pointer2 := 0 //point to tail

	for true {
		err := dbMgr.Db.View(func(txn *badger.Txn) error {
			key := nodeprefix + CHD_PREFIX + "_" + BLK_PREFIX + "_"
			opts := badger.DefaultIteratorOptions
			opts.PrefetchSize = 10
			it := txn.NewIterator(opts)
			defer it.Close()
			for it.Seek([]byte(key)); it.ValidForPrefix([]byte(key)); it.Next() {
				item := it.Item()
				err := item.Value(func(v []byte) error {
					contentitem := &quorumpb.BlockDbChunk{}
					ctnerr := proto.Unmarshal(v, contentitem)
					if ctnerr != nil {
						return ctnerr
					}

					if contentitem.BlockItem.PrevBlockId == blocks[pointer1].BlockId {
						blocks = append(blocks, contentitem.BlockItem)
						pointer2++
					}

					return nil
				})

				if err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			return blocks, err
		}

		if pointer1 == pointer2 {
			break
		}

		pointer1++
	}

	return blocks, nil
}

func (dbMgr *DbMgr) GetBlockHeight(blockId string, prefix ...string) (int64, error) {
	pChunk, err := dbMgr.getBlockChunk(blockId, false, prefix...)
	if err != nil {
		return -1, err
	}
	return pChunk.Height, nil
}

func (dbMgr *DbMgr) GetSubBlock(blockId string, prefix ...string) ([]*quorumpb.Block, error) {
	var result []*quorumpb.Block
	chunk, err := dbMgr.getBlockChunk(blockId, false, prefix...)
	if err != nil {
		return nil, err
	}

	for _, subChunkId := range chunk.SubBlockId {
		subChunk, err := dbMgr.getBlockChunk(subChunkId, false, prefix...)
		if err != nil {
			return nil, err
		}
		result = append(result, subChunk.BlockItem)
	}

	return result, nil
}

func (dbMgr *DbMgr) GetParentBlock(blockId string, prefix ...string) (*quorumpb.Block, error) {
	chunk, err := dbMgr.getBlockChunk(blockId, false, prefix...)
	if err != nil {
		return nil, err
	}

	parentChunk, err := dbMgr.getBlockChunk(chunk.ParentBlockId, false, prefix...)
	return parentChunk.BlockItem, err
}

//get block chunk
func (dbMgr *DbMgr) getBlockChunk(blockId string, cached bool, prefix ...string) (*quorumpb.BlockDbChunk, error) {
	nodeprefix := getPrefix(prefix...)
	var key string
	if cached {
		key = nodeprefix + CHD_PREFIX + "_" + BLK_PREFIX + "_" + blockId
	} else {
		key = nodeprefix + BLK_PREFIX + "_" + blockId
	}

	pChunk := &quorumpb.BlockDbChunk{}
	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		pBlockBytes, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		err = proto.Unmarshal(pBlockBytes, pChunk)
		return err
	})

	return pChunk, err
}

//save block chunk
func (dbMgr *DbMgr) saveBlockChunk(chunk *quorumpb.BlockDbChunk, cached bool, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
	var key string
	if cached {
		key = nodeprefix + CHD_PREFIX + "_" + BLK_PREFIX + "_" + chunk.BlockId
	} else {
		key = nodeprefix + BLK_PREFIX + "_" + chunk.BlockId
	}

	return dbMgr.Db.Update(func(txn *badger.Txn) error {
		bytes, err := proto.Marshal(chunk)
		e := badger.NewEntry([]byte(key), bytes)
		err = txn.SetEntry(e)
		return err
	})
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
	return dbMgr.GroupInfoDb.Update(func(txn *badger.Txn) error {
		bytes, err := proto.Marshal(groupItem)
		if err != nil {
			return err
		}
		e := badger.NewEntry([]byte(groupItem.GroupId), bytes)
		err = txn.SetEntry(e)
		return err
	})
}

func (dbMgr *DbMgr) UpdGroup(groupItem *quorumpb.GroupItem) error {
	//upd group to db
	return dbMgr.GroupInfoDb.Update(func(txn *badger.Txn) error {
		bytes, err := proto.Marshal(groupItem)
		if err != nil {
			return err
		}
		e := badger.NewEntry([]byte(groupItem.GroupId), bytes)
		err = txn.SetEntry(e)
		return err
	})
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
	return dbMgr.GroupInfoDb.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(item.GroupId))
		return err
	})
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

func (dbMgr *DbMgr) AddPost(trx *quorumpb.Trx, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + GRP_PREFIX + "_" + CNT_PREFIX + "_" + trx.GroupId + "_" + fmt.Sprint(trx.TimeStamp) + "_" + trx.TrxId
	dbmgr_log.Infof("Add POST with key %s", key)

	var ctnItem *quorumpb.PostItem
	ctnItem = &quorumpb.PostItem{}

	ctnItem.TrxId = trx.TrxId
	ctnItem.PublisherPubkey = trx.SenderPubkey
	ctnItem.Content = trx.Data
	ctnItem.TimeStamp = trx.TimeStamp
	ctnBytes, err := proto.Marshal(ctnItem)
	if err != nil {
		return err
	}

	return dbMgr.Db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(key), ctnBytes)
		err := txn.SetEntry(e)
		return err
	})
}

func (dbMgr *DbMgr) GetGrpCtnt(groupId string, ctntype string, prefix ...string) ([]*quorumpb.PostItem, error) {
	var ctnList []*quorumpb.PostItem
	nodeprefix := getPrefix(prefix...)
	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		key := nodeprefix + GRP_PREFIX + "_" + CNT_PREFIX + "_" + groupId + "_"
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

func (dbMgr *DbMgr) UpdateBlkListItem(trx *quorumpb.Trx, prefix ...string) (err error) {

	nodeprefix := getPrefix(prefix...)
	item := &quorumpb.DenyUserItem{}

	if err := proto.Unmarshal(trx.Data, item); err != nil {
		return err
	}

	if item.Action == "add" {
		key := nodeprefix + ATH_PREFIX + "_" + item.GroupId + "_" + item.PeerId
		err = dbMgr.Db.Update(func(txn *badger.Txn) error {
			e := badger.NewEntry([]byte(key), trx.Data)
			err := txn.SetEntry(e)
			return err
		})
	} else if item.Action == "del" {
		key := nodeprefix + ATH_PREFIX + "_" + item.GroupId + "_" + item.PeerId

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

func (dbMgr *DbMgr) GetBlkedUsers(prefix ...string) ([]*quorumpb.DenyUserItem, error) {
	var blkList []*quorumpb.DenyUserItem
	nodeprefix := getPrefix(prefix...)
	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		key := nodeprefix + ATH_PREFIX
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek([]byte(key)); it.ValidForPrefix([]byte(key)); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				blkItem := &quorumpb.DenyUserItem{}
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

func (dbMgr *DbMgr) IsUserBlocked(groupId, userId string, prefix ...string) (bool, error) {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + ATH_PREFIX + "_" + groupId + "_" + userId

	//check if group exist
	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(key))
		return err
	})

	if err == badger.ErrKeyNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func (dbMgr *DbMgr) UpdateProducer(trx *quorumpb.Trx, prefix ...string) (err error) {

	nodeprefix := getPrefix(prefix...)

	item := &quorumpb.ProducerItem{}
	if err := proto.Unmarshal(trx.Data, item); err != nil {
		return err
	}

	key := nodeprefix + PRD_PREFIX + "_" + item.GroupId + "_" + item.ProducerPubkey

	dbmgr_log.Infof("upd producer with key %s", key)

	if item.Action == "add" {
		dbmgr_log.Infof("Add producer")
		return dbMgr.Db.Update(func(txn *badger.Txn) error {
			e := badger.NewEntry([]byte(key), trx.Data)
			err := txn.SetEntry(e)
			return err
		})
	} else if item.Action == "remove" {
		//check if group exist
		dbmgr_log.Infof("Remove producer")
		err = dbMgr.Db.View(func(txn *badger.Txn) error {
			_, err := txn.Get([]byte(key))
			return err
		})

		if err == badger.ErrKeyNotFound {
			return errors.New("Producer not registed")
		} else if err != nil {
			return err
		}

		return dbMgr.Db.Update(func(txn *badger.Txn) error {
			err := txn.Delete([]byte(key))
			return err
		})
	} else {
		dbmgr_log.Infof("Remove producer")
		return errors.New("unknow msgType")
	}
}

func (dbMgr *DbMgr) AddProducer(item *quorumpb.ProducerItem, prefix ...string) error {

	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + PRD_PREFIX + "_" + item.GroupId + "_" + item.ProducerPubkey
	dbmgr_log.Infof("Add Producer with key %s", key)

	pbyte, err := proto.Marshal(item)
	if err != nil {
		return err
	}

	return dbMgr.Db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(key), pbyte)
		err := txn.SetEntry(e)
		return err
	})
}

func (dbMgr *DbMgr) GetProducers(groupId string, prefix ...string) ([]*quorumpb.ProducerItem, error) {
	var PrdList []*quorumpb.ProducerItem
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + PRD_PREFIX + "_" + groupId

	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek([]byte(key)); it.ValidForPrefix([]byte(key)); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				prdItem := &quorumpb.ProducerItem{}
				ctnerr := proto.Unmarshal(v, prdItem)
				if ctnerr == nil {
					PrdList = append(PrdList, prdItem)
				}
				return ctnerr
			})

			if err != nil {
				return err
			}
		}
		return nil
	})

	return PrdList, err
}

func (dbMgr *DbMgr) IsProducer(groupId, producerPubKey string, prefix ...string) (bool, error) {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + PRD_PREFIX + "_" + groupId + "_" + producerPubKey

	//check if group exist
	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(key))
		return err
	})

	if err == badger.ErrKeyNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func (dbMgr *DbMgr) UpdateAnnounce(trx *quorumpb.Trx, prefix ...string) (err error) {

	nodeprefix := getPrefix(prefix...)
	item := &quorumpb.AnnounceItem{}
	if err := proto.Unmarshal(trx.Data, item); err != nil {
		return err
	}
	key := nodeprefix + ANN_PREFIX + "_" + item.GroupId + "_" + item.Type + "_" + item.AnnouncedPubkey

	if item.Action == "add" {
		err = dbMgr.Db.Update(func(txn *badger.Txn) error {
			e := badger.NewEntry([]byte(key), trx.Data)
			err := txn.SetEntry(e)
			return err
		})
	} else if item.Action == "del" {
		//check if item exist
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

func (dbMgr *DbMgr) GetAnnouncedUsers(groupId string, prefix ...string) ([]*quorumpb.AnnounceItem, error) {
	var AnnList []*quorumpb.AnnounceItem

	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + ANN_PREFIX + "_" + groupId + "_" + "userpubkey" // announced type is "userpubkey"
	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek([]byte(key)); it.ValidForPrefix([]byte(key)); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				annItem := &quorumpb.AnnounceItem{}
				ctnerr := proto.Unmarshal(v, annItem)
				if ctnerr == nil {
					AnnList = append(AnnList, annItem)
				}

				return ctnerr
			})

			if err != nil {
				return err
			}
		}
		return nil
	})

	return AnnList, err
}

func (dbMgr *DbMgr) IsUser(groupId, userPubKey string, prefix ...string) (bool, error) {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + ANN_PREFIX + "_" + groupId + "_" + userPubKey

	//check if group user (announced) exist
	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(key))
		return err
	})

	if err == badger.ErrKeyNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func (dbMgr *DbMgr) UpdateSchema(trx *quorumpb.Trx, prefix ...string) (err error) {
	item := &quorumpb.SchemaItem{}
	if err := proto.Unmarshal(trx.Data, item); err != nil {
		return err
	}

	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + SMA_PREFIX + "_" + item.GroupId + "_" + item.SchemaJson

	if item.Memo == "Add" || item.Memo == "Update" {
		err = dbMgr.Db.Update(func(txn *badger.Txn) error {
			e := badger.NewEntry([]byte(key), trx.Data)
			err := txn.SetEntry(e)
			return err
		})
	} else if item.Memo == "Remove" {
		//check if item exist
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

func (dbMgr *DbMgr) GetSchemaByGroup(groupId string, prefix ...string) ([]*quorumpb.SchemaItem, error) {
	var scmList []*quorumpb.SchemaItem

	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + SMA_PREFIX + "_" + groupId
	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek([]byte(key)); it.ValidForPrefix([]byte(key)); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				scmItem := &quorumpb.SchemaItem{}
				ctnerr := proto.Unmarshal(v, scmItem)
				if ctnerr == nil {
					scmList = append(scmList, scmItem)
				}

				return ctnerr
			})

			if err != nil {
				return err
			}
		}
		return nil
	})

	return scmList, err
}

func getPrefix(prefix ...string) string {
	nodeprefix := ""
	if len(prefix) == 1 {
		nodeprefix = prefix[0] + "_"
	}
	return nodeprefix
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

/*
func (dbMgr *DbMgr) IsAnnouncedGroupProducer(groupId, pubKey string) (bool, error) {
	key := ANN_PREFIX + "_" + groupId + "_Producer" + "_" + pubKey

	//check if group producer (announce) exist
	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(key))
		return err
	})

	if err == nil {
		return true, nil
	}

	return false, err
}
*/

/*
func (dbMgr *DbMgr) GetAnnounceProducersByGroup(groupId string) ([]*quorumpb.AnnounceItem, error) {
	var AnnList []*quorumpb.AnnounceItem
	key := ANN_PREFIX + groupId + "_Producer"

	err := dbMgr.Db.View(func(txn *badger.Txn) error {
		fmt.Println(key)
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek([]byte(key)); it.ValidForPrefix([]byte(key)); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				annItem := &quorumpb.AnnounceItem{}
				ctnerr := proto.Unmarshal(v, annItem)
				if ctnerr == nil {
					AnnList = append(AnnList, annItem)
				}

				return ctnerr
			})

			if err != nil {
				return err
			}
		}
		return nil
	})

	return AnnList, err
}
*/
