package chainstorage

import (
	s "github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

//add block
func (cs *Storage) AddBlock(newBlock *quorumpb.Block, cached bool, prefix ...string) error {
	isSaved, err := cs.dbmgr.IsBlockExist(newBlock.BlockId, cached, prefix...)
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
	return cs.dbmgr.SaveBlockChunk(chunk, cached, prefix...)
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

	return cs.dbmgr.Db.Delete([]byte(key))
}

//get block by block_id
func (cs *Storage) GetBlock(blockId string, cached bool, prefix ...string) (*quorumpb.Block, error) {
	pChunk, err := cs.dbmgr.GetBlockChunk(blockId, cached, prefix...)
	if err != nil {
		return nil, err
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
