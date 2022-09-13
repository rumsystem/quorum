package chainstorage

import (
	"fmt"
	"log"
	"os"

	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	s "github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

//add block
func (cs *Storage) AddBlock(newBlock *quorumpb.Block, cached bool, prefix ...string) error {
	isSaved, err := cs.IsBlockExist(newBlock.BlockId, cached, prefix...)
	if err != nil {
		return err
	}
	if isSaved {
		chaindb_log.Debugf("Block <%s> already saved, ignore", newBlock.BlockId)
		return nil
	}

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
		pChunk, err := cs.dbmgr.GetBlockChunk(newBlock.PrevBlockId, cached, prefix...)
		if err != nil {
			return err
		}

		//update parent chunk
		pChunk.SubBlockId = append(pChunk.SubBlockId, chunk.BlockId)
		err = cs.dbmgr.SaveBlockChunk(pChunk, cached, prefix...)
		if err != nil {
			return err
		}

		chunk.Height = pChunk.Height + 1     //increase height
		chunk.ParentBlockId = pChunk.BlockId //point to parent
	}

	//save chunk
	err = cs.dbmgr.SaveBlockChunk(chunk, cached, prefix...)
	return err
}

//remove block
func (cs *Storage) RmBlock(blockId string, cached bool, prefix ...string) error {
	nodeprefix := utils.GetPrefix(prefix...)
	var key string
	if cached {
		key = nodeprefix + s.CHD_PREFIX + "_" + s.BLK_PREFIX + "_" + blockId
	} else {
		key = nodeprefix + s.BLK_PREFIX + "_" + blockId
	}

	err := cs.dbmgr.Db.Delete([]byte(key))
	return err
}

//get block by block_id
func (cs *Storage) GetBlock(blockId string, cached bool, prefix ...string) (*quorumpb.Block, error) {
	pChunk, err := cs.dbmgr.GetBlockChunk(blockId, cached, prefix...)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(pChunk.BlockItem.Trxs); i++ {
		pk, err := localcrypto.Libp2pPubkeyToEthBase64(pChunk.BlockItem.Trxs[i].SenderPubkey)
		if err == nil {
			pChunk.BlockItem.Trxs[i].SenderPubkey = pk
		}
	}
	return pChunk.BlockItem, nil
}

func (cs *Storage) GatherBlocksFromCache(newBlock *quorumpb.Block, cached bool, prefix ...string) ([]*quorumpb.Block, error) {
	nodeprefix := utils.GetPrefix(prefix...)
	var blocks []*quorumpb.Block
	blocks = append(blocks, newBlock)
	pointer1 := 0 //point to head
	pointer2 := 0 //point to tail

	pre := nodeprefix + s.CHD_PREFIX + "_" + s.BLK_PREFIX + "_"

	for {
		err := cs.dbmgr.Db.PrefixForeach([]byte(pre), func(k []byte, v []byte, err error) error {
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

func (cs *Storage) AddGensisBlock(gensisBlock *quorumpb.Block, prefix ...string) error {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.BLK_PREFIX + "_" + gensisBlock.BlockId

	isExist, err := cs.dbmgr.Db.IsExist([]byte(key))
	if err != nil {
		return err
	}
	if isExist {
		chaindb_log.Debugf("Genesis block <%s> exist, do nothing", gensisBlock.BlockId)
		return nil
	}

	chunk := quorumpb.BlockDbChunk{}
	chunk.BlockId = gensisBlock.BlockId
	chunk.BlockItem = gensisBlock
	chunk.ParentBlockId = ""
	chunk.Height = 0

	value, err := proto.Marshal(&chunk)
	if err != nil {
		return err
	}

	return cs.dbmgr.Db.Set([]byte(key), value)
}

func (cs *Storage) GetBlockHeight(blockId string, prefix ...string) (int64, error) {

	pChunk, err := cs.dbmgr.GetBlockChunk(blockId, false, prefix...)
	if err != nil {
		return -1, err
	}
	return pChunk.Height, nil
}

//check if block existed
func (cs *Storage) IsBlockExist(blockId string, cached bool, prefix ...string) (bool, error) {
	nodeprefix := utils.GetPrefix(prefix...)
	var key string
	if cached {
		key = nodeprefix + s.CHD_PREFIX + "_" + s.BLK_PREFIX + "_" + blockId
	} else {
		key = nodeprefix + s.BLK_PREFIX + "_" + blockId
	}

	r, err := cs.dbmgr.Db.IsExist([]byte(key))
	return r, err
}

//check if parent block existed
func (cs *Storage) IsParentExist(parentBlockId string, cached bool, prefix ...string) (bool, error) {
	nodeprefix := utils.GetPrefix(prefix...)
	var pKey string
	if cached {
		pKey = nodeprefix + s.CHD_PREFIX + "_" + s.BLK_PREFIX + "_" + parentBlockId
	} else {
		pKey = nodeprefix + s.BLK_PREFIX + "_" + parentBlockId
	}

	return cs.dbmgr.Db.IsExist([]byte(pKey))
}

func (cs *Storage) GetSubBlock(blockId string, prefix ...string) ([]*quorumpb.Block, error) {
	var result []*quorumpb.Block
	chunk, err := cs.dbmgr.GetBlockChunk(blockId, false, prefix...)
	if err != nil {
		return nil, err
	}

	for _, subChunkId := range chunk.SubBlockId {
		subChunk, err := cs.dbmgr.GetBlockChunk(subChunkId, false, prefix...)
		if err != nil {
			return nil, err
		}
		result = append(result, subChunk.BlockItem)
	}

	return result, nil
}

func (cs *Storage) GetParentBlock(blockId string, prefix ...string) (*quorumpb.Block, error) {
	chunk, err := cs.dbmgr.GetBlockChunk(blockId, false, prefix...)
	if err != nil {
		return nil, err
	}
	parentChunk, err := cs.dbmgr.GetBlockChunk(chunk.ParentBlockId, false, prefix...)
	if err == nil {
		return parentChunk.BlockItem, err
	} else {
		return nil, err
	}
}

//try to find the subblocks of one block. search from the block to the to blockid
func (cs *Storage) RepairSubblocksList(blockid, toblockid string, prefix ...string) error {
	if toblockid == blockid {
		return fmt.Errorf("no new blocks, no need to repair")
	}
	blockChunk, err := cs.dbmgr.GetBlockChunk(blockid, false, prefix...)
	if err != nil {
		return err
	}
	blockChunk.SubBlockId = []string{}
	succ := false
	verifyblockid := toblockid
	var dblogger *log.Logger
	logfile, err := os.OpenFile(cs.dbmgr.DataPath+"_blockdbrepair.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer logfile.Close()
	if err != nil {
	}
	dblogger = log.New(logfile, "blockdb", log.LstdFlags)

	dblogger.Printf("verify block: %s", blockid)
	var verifyblockChunk *quorumpb.BlockDbChunk
	for {
		verifyblockChunk, err = cs.dbmgr.GetBlockChunk(verifyblockid, false, prefix...)
		if verifyblockid == blockid {
			break
		}
		if verifyblockChunk == nil {
			dblogger.Printf("the block is nil, id: %s", verifyblockid)
			break
		}
		if verifyblockChunk.ParentBlockId == blockid {
			blockChunk.SubBlockId = append(blockChunk.SubBlockId, verifyblockChunk.BlockId)
			dblogger.Printf("find the subblock of %s the subblockid is %s", blockid, verifyblockChunk.BlockId)
			succ = true
			break
		}
		verifyblockid = verifyblockChunk.ParentBlockId
	}
	if succ == false {
		dblogger.Printf("not find the subblock of %s", blockid)
	} else {
		dblogger.Printf("update the subblockid of %s", blockid)
		err = cs.dbmgr.SaveBlockChunk(blockChunk, false, prefix...)
		if err != nil {
			dblogger.Printf("Error: %s", err)
		}
	}
	return nil
}
