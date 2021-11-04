package storage

import (
	"errors"
	"fmt"

	logging "github.com/ipfs/go-log/v2"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var dbmgr_log = logging.Logger("dbmgr")

const TRX_PREFIX string = "trx" //trx
const BLK_PREFIX string = "blk" //block
const GRP_PREFIX string = "grp" //group
const CNT_PREFIX string = "cnt" //content
const ATH_PREFIX string = "ath" //auth
const PRD_PREFIX string = "prd" //producer
const ANN_PREFIX string = "ann" //announce
const SMA_PREFIX string = "sma" //schema
const CHD_PREFIX string = "chd" //cached

type DbMgr struct {
	GroupInfoDb QuorumStorage
	Db          QuorumStorage
	Auth        QuorumStorage
	DataPath    string
}

func (dbMgr *DbMgr) CloseDb() {
	dbMgr.GroupInfoDb.Close()
	dbMgr.Db.Close()
	dbmgr_log.Infof("ChainCtx Db closed")
}

func (dbMgr *DbMgr) TryMigration(nodeDataVer int) {
	if nodeDataVer == 0 { //try migration 0 (Upgrade the GroupItem)

		dbmgr_log.Infof("db migration v0")
		groupItemsBytes, err := dbMgr.GetGroupsBytes()
		if err == nil {
			for _, b := range groupItemsBytes {
				var item *quorumpb.GroupItem
				item = &quorumpb.GroupItem{}
				proto.Unmarshal(b, item)
				if item.CipherKey == "" {
					itemv0 := &quorumpb.GroupItemV0{}
					proto.Unmarshal(b, itemv0)
					if itemv0.CipherKey != "" { //ok
						item.LastUpdate = itemv0.LastUpdate
						item.HighestHeight = itemv0.HighestHeight
						item.HighestBlockId = itemv0.HighestBlockId
						item.GenesisBlock = itemv0.GenesisBlock
						item.EncryptType = itemv0.EncryptType
						item.ConsenseType = itemv0.ConsenseType
						item.CipherKey = itemv0.CipherKey
						item.AppKey = itemv0.AppKey
						//add group to db
						value, err := proto.Marshal(item)
						if err == nil {
							dbMgr.GroupInfoDb.Set([]byte(item.GroupId), value)
							dbmgr_log.Infof("db migration v0 for group %s", item.GroupId)
						}
					}
				}
			}

		}

	}

}

//save trx
func (dbMgr *DbMgr) AddTrx(trx *quorumpb.Trx, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + TRX_PREFIX + "_" + trx.TrxId
	value, err := proto.Marshal(trx)
	if err != nil {
		return err
	}
	return dbMgr.Db.Set([]byte(key), value)
}

//UNUSED
//rm Trx
func (dbMgr *DbMgr) RmTrx(trxId string, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + TRX_PREFIX + "_" + trxId
	return dbMgr.Db.Delete([]byte(key))
}

//get trx
func (dbMgr *DbMgr) GetTrx(trxId string, prefix ...string) (*quorumpb.Trx, error) {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + TRX_PREFIX + "_" + trxId
	value, err := dbMgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	var trx quorumpb.Trx
	err = proto.Unmarshal(value, &trx)
	if err != nil {
		return nil, err
	}

	return &trx, err
}

func (dbMgr *DbMgr) UpdTrx(trx *quorumpb.Trx, prefix ...string) error {
	return dbMgr.AddTrx(trx, prefix...)
}

func (dbMgr *DbMgr) IsTrxExist(trxId string, prefix ...string) (bool, error) {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + TRX_PREFIX + "_" + trxId

	return dbMgr.Db.IsExist([]byte(key))
}

func (dbMgr *DbMgr) AddGensisBlock(gensisBlock *quorumpb.Block, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + BLK_PREFIX + "_" + gensisBlock.BlockId

	chunk := quorumpb.BlockDbChunk{}
	chunk.BlockId = gensisBlock.BlockId
	chunk.BlockItem = gensisBlock
	chunk.ParentBlockId = ""
	chunk.Height = 0

	value, err := proto.Marshal(&chunk)
	if err != nil {
		return err
	}

	return dbMgr.Db.Set([]byte(key), value)
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

	return dbMgr.Db.IsExist([]byte(key))
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

	return dbMgr.Db.IsExist([]byte(pKey))
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

	return dbMgr.Db.Delete([]byte(key))
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

	pre := nodeprefix + CHD_PREFIX + "_" + BLK_PREFIX + "_"

	for {
		err := dbMgr.Db.PrefixForeach([]byte(pre), func(k []byte, v []byte, err error) error {
			if err != nil {
				return err
			}
			chunk := quorumpb.BlockDbChunk{}
			perr := proto.Unmarshal(v, &chunk)
			if perr != nil {
				return perr
			}
			if chunk.BlockItem.PrevBlockId == blocks[pointer1].BlockId {
				blocks = append(blocks, chunk.BlockItem)
				pointer2++
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

	pChunk := quorumpb.BlockDbChunk{}
	value, err := dbMgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	err = proto.Unmarshal(value, &pChunk)
	if err != nil {
		return nil, err
	}

	return &pChunk, err
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

	value, err := proto.Marshal(chunk)
	if err != nil {
		return err
	}
	return dbMgr.Db.Set([]byte(key), value)
}

func (dbMgr *DbMgr) AddGroup(groupItem *quorumpb.GroupItem) error {
	//check if group exist
	exist, err := dbMgr.GroupInfoDb.IsExist([]byte(groupItem.GroupId))
	if exist {
		return errors.New("Group with same GroupId existed")
	}

	//add group to db
	value, err := proto.Marshal(groupItem)
	if err != nil {
		return err
	}
	return dbMgr.GroupInfoDb.Set([]byte(groupItem.GroupId), value)
}

func (dbMgr *DbMgr) UpdGroup(groupItem *quorumpb.GroupItem) error {
	value, err := proto.Marshal(groupItem)
	if err != nil {
		return err
	}

	//upd group to db
	return dbMgr.GroupInfoDb.Set([]byte(groupItem.GroupId), value)
}

func (dbMgr *DbMgr) RmGroup(item *quorumpb.GroupItem) error {
	//check if group exist
	exist, err := dbMgr.GroupInfoDb.IsExist([]byte(item.GroupId))
	if !exist {
		if err != nil {
			return err
		}
		return errors.New("Group Not Found")
	}

	//delete group
	return dbMgr.GroupInfoDb.Delete([]byte(item.GroupId))
}

func (dbMgr *DbMgr) RemoveGroupData(item *quorumpb.GroupItem, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
	var keys []string

	//remove all group POST
	key := nodeprefix + GRP_PREFIX + "_" + CNT_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//all group producer
	key = nodeprefix + PRD_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//all group block list
	key = nodeprefix + ATH_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//all group announced item
	key = nodeprefix + ANN_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//all group schema item
	key = nodeprefix + SMA_PREFIX + "_" + item.GroupId
	keys = append(keys, key)

	//remove all
	for _, key_prefix := range keys {
		err := dbMgr.Db.PrefixForeachKey([]byte(key_prefix), []byte(key_prefix), false, func(k []byte, err error) error {
			if err != nil {
				return err
			}
			dbmgr_log.Debugf("Remove key %s", string(k))
			return dbMgr.Db.Delete(k)
		})

		if err != nil {

			return err
		}
	}

	keys = nil
	//remove all cached block
	key = nodeprefix + BLK_PREFIX + "_"
	keys = append(keys, key)
	key = nodeprefix + CHD_PREFIX + "_" + BLK_PREFIX + "_"
	keys = append(keys, key)

	for _, key_prefix := range keys {
		err := dbMgr.Db.PrefixForeach([]byte(key_prefix), func(k []byte, v []byte, err error) error {
			if err != nil {
				return err
			}

			blockChunk := quorumpb.BlockDbChunk{}
			perr := proto.Unmarshal(v, &blockChunk)
			if perr != nil {
				return perr
			}

			if blockChunk.BlockItem.GroupId == item.GroupId {
				dbmgr_log.Debugf("Remove key %s", string(k))
				return dbMgr.Db.Delete(k)
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	//remove all trx
	key = nodeprefix + TRX_PREFIX + "_"
	err := dbMgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}

		trx := quorumpb.Trx{}
		perr := proto.Unmarshal(v, &trx)

		if perr != nil {
			return perr
		}

		if trx.GroupId == item.GroupId {
			dbmgr_log.Debugf("Remove key %s", string(k))
			return dbMgr.Db.Delete(k)
		}

		return nil
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
	err := dbMgr.GroupInfoDb.Foreach(func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		groupItemList = append(groupItemList, v)
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

	return dbMgr.Db.Set([]byte(key), ctnBytes)
}

func (dbMgr *DbMgr) GetGrpCtnt(groupId string, ctntype string, prefix ...string) ([]*quorumpb.PostItem, error) {
	var ctnList []*quorumpb.PostItem
	nodeprefix := getPrefix(prefix...)
	pre := nodeprefix + GRP_PREFIX + "_" + CNT_PREFIX + "_" + groupId + "_"
	err := dbMgr.Db.PrefixForeach([]byte(pre), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}

		item := quorumpb.PostItem{}
		perr := proto.Unmarshal(v, &item)
		if perr != nil {
			return perr
		}
		ctnList = append(ctnList, &item)
		return nil
	})

	return ctnList, err
}

//func (dbMgr *DbMgr) GetTrxContent(trxId string, prefix ...string) (*quorumpb.Trx, error) {
//	nodeprefix := getPrefix(prefix...)
//	var trx quorumpb.Trx
//	key := nodeprefix + TRX_PREFIX + "_" + trxId
//	err := dbMgr.Db.View(func(txn *badger.Txn) error {
//		item, err := txn.Get([]byte(key))
//		if err != nil {
//			return err
//		}
//
//		trxBytes, err := item.ValueCopy(nil)
//		if err != nil {
//			return err
//		}
//
//		err = proto.Unmarshal(trxBytes, &trx)
//		if err != nil {
//			return err
//		}
//
//		return nil
//	})
//
//	return &trx, err
//}

func (dbMgr *DbMgr) UpdateBlkListItem(trx *quorumpb.Trx, prefix ...string) (err error) {
	nodeprefix := getPrefix(prefix...)
	item := &quorumpb.DenyUserItem{}

	if err := proto.Unmarshal(trx.Data, item); err != nil {
		return err
	}

	if item.Action == "add" {
		key := nodeprefix + ATH_PREFIX + "_" + item.GroupId + "_" + item.PeerId
		return dbMgr.Db.Set([]byte(key), trx.Data)
	} else if item.Action == "del" {
		key := nodeprefix + ATH_PREFIX + "_" + item.GroupId + "_" + item.PeerId

		//check if group exist
		exist, err := dbMgr.Db.IsExist([]byte(key))
		if !exist {
			if err != nil {
				return err
			}
			return errors.New("Group Not Found")
		}
		return dbMgr.Db.Delete([]byte(key))
	} else {
		return errors.New("unknow msgType")
	}
}

func (dbMgr *DbMgr) GetBlkedUsers(prefix ...string) ([]*quorumpb.DenyUserItem, error) {
	var blkList []*quorumpb.DenyUserItem
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + ATH_PREFIX
	err := dbMgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}

		item := quorumpb.DenyUserItem{}
		perr := proto.Unmarshal(v, &item)
		if perr != nil {
			return perr
		}
		blkList = append(blkList, &item)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return blkList, nil
}

func (dbMgr *DbMgr) IsUserBlocked(groupId, userId string, prefix ...string) (bool, error) {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + ATH_PREFIX + "_" + groupId + "_" + userId

	//check if group exist
	return dbMgr.Db.IsExist([]byte(key))
}

func (dbMgr *DbMgr) UpdateProducer(trx *quorumpb.Trx, prefix ...string) (err error) {

	nodeprefix := getPrefix(prefix...)

	item := &quorumpb.ProducerItem{}
	if err := proto.Unmarshal(trx.Data, item); err != nil {
		return err
	}

	key := nodeprefix + PRD_PREFIX + "_" + item.GroupId + "_" + item.ProducerPubkey

	dbmgr_log.Infof("upd producer with key %s", key)

	if item.Action == quorumpb.ActionType_ADD {
		dbmgr_log.Infof("Add producer")
		return dbMgr.Db.Set([]byte(key), trx.Data)
	} else if item.Action == quorumpb.ActionType_REMOVE {
		//check if group exist
		dbmgr_log.Infof("Remove producer")
		exist, err := dbMgr.Db.IsExist([]byte(key))
		if !exist {
			if err != nil {
				return err
			}
			return errors.New("Producer Not Found")
		}

		return dbMgr.Db.Delete([]byte(key))
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
	return dbMgr.Db.Set([]byte(key), pbyte)
}

func (dbMgr *DbMgr) AddProducedBlockCount(groupId, producerPubkey string, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + PRD_PREFIX + "_" + groupId + "_" + producerPubkey
	var pProducer *quorumpb.ProducerItem
	pProducer = &quorumpb.ProducerItem{}

	value, err := dbMgr.Db.Get([]byte(key))
	if err != nil {
		return err
	}

	err = proto.Unmarshal(value, pProducer)
	if err != nil {
		return err
	}

	pProducer.BlockProduced += 1

	value, err = proto.Marshal(pProducer)
	if err != nil {
		return err
	}
	return dbMgr.Db.Set([]byte(key), value)
}

func (dbMgr *DbMgr) GetProducers(groupId string, prefix ...string) ([]*quorumpb.ProducerItem, error) {
	var pList []*quorumpb.ProducerItem
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + PRD_PREFIX + "_" + groupId

	err := dbMgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		item := quorumpb.ProducerItem{}
		perr := proto.Unmarshal(v, &item)
		if perr != nil {
			return perr
		}
		pList = append(pList, &item)
		return nil
	})
	return pList, err
}

func (dbMgr *DbMgr) IsProducer(groupId, producerPubKey string, prefix ...string) (bool, error) {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + PRD_PREFIX + "_" + groupId + "_" + producerPubKey

	//check if group exist
	return dbMgr.Db.IsExist([]byte(key))
}

func (dbMgr *DbMgr) UpdateAnnounce(trx *quorumpb.Trx, prefix ...string) (err error) {

	nodeprefix := getPrefix(prefix...)
	item := &quorumpb.AnnounceItem{}
	if err := proto.Unmarshal(trx.Data, item); err != nil {
		return err
	}
	key := nodeprefix + ANN_PREFIX + "_" + item.GroupId + "_" + item.Type.Enum().String() + "_" + item.SignPubkey
	return dbMgr.Db.Set([]byte(key), trx.Data)
}

func (dbMgr *DbMgr) GetAnnouncedUsersByGroup(groupId string, prefix ...string) ([]*quorumpb.AnnounceItem, error) {
	var aList []*quorumpb.AnnounceItem

	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_USER.String()
	err := dbMgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		item := quorumpb.AnnounceItem{}
		perr := proto.Unmarshal(v, &item)
		if perr != nil {
			return perr
		}
		aList = append(aList, &item)
		return nil
	})

	return aList, err
}

func (dbMgr *DbMgr) GetAnnounceProducersByGroup(groupId string, prefix ...string) ([]*quorumpb.AnnounceItem, error) {
	var aList []*quorumpb.AnnounceItem

	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_PRODUCER.String()
	err := dbMgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		item := quorumpb.AnnounceItem{}
		perr := proto.Unmarshal(v, &item)
		if perr != nil {
			return perr
		}
		aList = append(aList, &item)
		return nil
	})

	return aList, err
}

func (dbMgr *DbMgr) GetAnnouncedProducer(groupId string, pubkey string, prefix ...string) (*quorumpb.AnnounceItem, error) {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_PRODUCER.String() + "_" + pubkey

	value, err := dbMgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	var ann quorumpb.AnnounceItem
	err = proto.Unmarshal(value, &ann)
	if err != nil {
		return nil, err
	}

	return &ann, err
}

func (dbMgr *DbMgr) IsProducerAnnounced(groupId, producerSignPubkey string, prefix ...string) (bool, error) {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_PRODUCER.String() + "_" + producerSignPubkey
	return dbMgr.Db.IsExist([]byte(key))
}

func (dbMgr *DbMgr) UpdateProducerAnnounceResult(groupId, producerSignPubkey string, result bool, prefix ...string) error {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_PRODUCER.String() + "_" + producerSignPubkey

	var pAnnounced *quorumpb.AnnounceItem
	pAnnounced = &quorumpb.AnnounceItem{}

	value, err := dbMgr.Db.Get([]byte(key))
	if err != nil {
		return err
	}

	err = proto.Unmarshal(value, pAnnounced)
	if err != nil {
		return err
	}

	if result {
		pAnnounced.Result = quorumpb.ApproveType_APPROVED
	} else {
		pAnnounced.Result = quorumpb.ApproveType_ANNOUCNED
	}

	value, err = proto.Marshal(pAnnounced)
	if err != nil {
		return err
	}
	return dbMgr.Db.Set([]byte(key), value)
}

func (dbMgr *DbMgr) IsUser(groupId, userPubKey string, prefix ...string) (bool, error) {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + ANN_PREFIX + "_" + groupId + "_" + quorumpb.AnnounceType_AS_USER.String() + "_" + userPubKey

	//check if group user (announced) exist
	return dbMgr.Db.IsExist([]byte(key))
}

func (dbMgr *DbMgr) UpdateSchema(trx *quorumpb.Trx, prefix ...string) (err error) {
	item := &quorumpb.SchemaItem{}
	if err := proto.Unmarshal(trx.Data, item); err != nil {
		return err
	}

	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + SMA_PREFIX + "_" + item.GroupId + "_" + item.Type

	if item.Action == quorumpb.ActionType_ADD {
		return dbMgr.Db.Set([]byte(key), trx.Data)
	} else if item.Action == quorumpb.ActionType_REMOVE {
		//check if item exist
		exist, err := dbMgr.Db.IsExist([]byte(key))
		if !exist {
			if err != nil {
				return err
			}
			return errors.New("Announce Not Found")
		}

		return dbMgr.Db.Delete([]byte(key))
	} else {
		err := errors.New("unknow msgType")
		return err
	}
}

func (dbMgr *DbMgr) GetAllSchemasByGroup(groupId string, prefix ...string) ([]*quorumpb.SchemaItem, error) {
	var scmList []*quorumpb.SchemaItem

	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + SMA_PREFIX + "_" + groupId

	err := dbMgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		item := quorumpb.SchemaItem{}
		perr := proto.Unmarshal(v, &item)
		if perr != nil {
			return perr
		}
		scmList = append(scmList, &item)
		return nil
	})

	return scmList, err
}

func (dbMgr *DbMgr) GetSchemaByGroup(groupId, schemaType string, prefix ...string) (*quorumpb.SchemaItem, error) {
	nodeprefix := getPrefix(prefix...)
	key := nodeprefix + SMA_PREFIX + "_" + groupId + "_" + schemaType

	schema := quorumpb.SchemaItem{}
	value, err := dbMgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	err = proto.Unmarshal(value, &schema)
	if err != nil {
		return nil, err
	}

	return &schema, err
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
