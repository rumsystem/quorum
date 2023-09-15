package chainstorage

import (
	"errors"
	"fmt"

	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	s "github.com/rumsystem/quorum/internal/pkg/storage"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

// add block
func (cs *Storage) AddBlock(block *quorumpb.Block, cached bool, prefix ...string) error {
	return cs.dbmgr.SaveBlock(block, cached, prefix...)
}

// add genesis block
func (cs *Storage) AddGensisBlock(block *quorumpb.Block, cached bool, prefix ...string) error {
	err := cs.dbmgr.SaveBlock(block, cached, prefix...)
	if err == rumerrors.ErrBlockExist {
		return nil
	}
	return err
}

// save block to cache datasource
func (cs *Storage) AddBlockToDSCache(block *quorumpb.Block, prefix ...string) error {
	fmt.Println("=======TODO: check if block exist")
	return cs.dbmgr.SaveBlockToDSCache(block, prefix...)
}

// get block by block_id
func (cs *Storage) GetBlockFromDSCache(groupId string, blockId uint64, prefix ...string) (*quorumpb.Block, error) {
	return cs.dbmgr.GetBlockFromDSCache(groupId, blockId, prefix...)
}

// remove block
func (cs *Storage) RmBlock(groupId string, blockId uint64, cached bool, prefix ...string) error {
	return cs.dbmgr.RmBlock(groupId, blockId, cached, prefix...)
}

// get block by block_id
func (cs *Storage) GetBlock(groupId string, blockId uint64, cached bool, prefix ...string) (*quorumpb.Block, error) {
	var key string
	if cached {
		key = s.GetCachedBlockKey(groupId, blockId, prefix...)
	} else {
		key = s.GetBlockKey(groupId, blockId, prefix...)
	}

	isExist, err := cs.dbmgr.Db.IsExist([]byte(key))
	if err != nil {
		return nil, err
	}

	if !isExist {
		return nil, rumerrors.ErrBlockExist
	}

	value, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}
	block := quorumpb.Block{}
	err = proto.Unmarshal(value, &block)
	if err != nil {
		return nil, err
	}

	return &block, err
}

// check if block exist
func (cs *Storage) IsBlockExist(groupId string, blockId uint64, cached bool, prefix ...string) (bool, error) {
	return cs.dbmgr.IsBlockExist(groupId, blockId, cached, prefix...)
}

func (cs *Storage) GatherBlocksFromCache(block *quorumpb.Block, prefix ...string) ([]*quorumpb.Block, error) {
	var blocks []*quorumpb.Block
	blocks = append(blocks, block)
	currBlockId := block.BlockId
	pre := s.GetCachedBlockPrefix(block.GroupId, prefix...)
	err := cs.dbmgr.Db.PrefixForeach([]byte(pre), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}

		b := &quorumpb.Block{}
		perr := proto.Unmarshal(v, b)
		if perr != nil {
			return perr
		}

		currBlockId += 1
		if b.GroupId == block.GroupId && b.BlockId == currBlockId {
			blocks = append(blocks, b)
			return nil
		} else {
			return errors.New("NO_MORE_BLOCK")
		}
	})

	//search done, no more block to attach
	if err == nil || err.Error() == "NO_MORE_BLOCK" {
		return blocks, nil
	}

	return nil, err
}

func (cs *Storage) GetNextConsensusNonce(groupId string, prefix ...string) (uint64, error) {
	return cs.dbmgr.GetNextConsensusNonce(groupId, prefix...)
}
