package chain

import (
	"bytes"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

//find the highest block from the block tree
func (chain *Chain) RecalChainHeight(blocks []*quorumpb.Block, currentHeight int64, currentHighestBlock *quorumpb.Block, nodename string) (int64, string, error) {
	newHighestHeight := currentHeight
	newHighestBlockId := currentHighestBlock.BlockId
	newHighestBlock := currentHighestBlock

	for _, block := range blocks {
		blockHeight, err := nodectx.GetNodeCtx().GetChainStorage().GetBlockHeight(block.BlockId, nodename)
		if err != nil {
			return -1, "INVALID_BLOCK_ID", err
		}
		if blockHeight > newHighestHeight {
			newHighestHeight = blockHeight
			newHighestBlockId = block.BlockId
			newHighestBlock = block
		} else if blockHeight == newHighestHeight {
			//comparing two hash bytes lexicographicall
			if bytes.Compare(newHighestBlock.Hash[:], block.Hash[:]) == -1 { //-1 means ohash < nhash, and we want keep the larger one
				newHighestHeight = blockHeight
				newHighestBlockId = block.BlockId
				newHighestBlock = block
			}
		}
	}

	return newHighestHeight, newHighestBlockId, nil
}

//from root of the new block tree, get all blocks trimed when not belong to longest path
func (chain *Chain) GetTrimedBlocks(blocks []*quorumpb.Block, nodename string) ([]string, error) {
	var cache map[string]bool
	var longestPath []string
	var result []string

	cache = make(map[string]bool)

	err := dfs(blocks, cache, longestPath, nodename)

	for _, blockId := range longestPath {
		if _, ok := cache[blockId]; !ok {
			result = append(result, blockId)
		}
	}

	return result, err
}

//get all trx belongs to me from the block list
func (chain *Chain) GetMyTrxs(blockIds []string, nodename string, userSignPubkey string) ([]*quorumpb.Trx, error) {
	var trxs []*quorumpb.Trx

	for _, blockId := range blockIds {
		block, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(blockId, false, nodename)
		if err != nil {
			//chain_log.Warnf(err.Error())
			continue
		}

		for _, trx := range block.Trxs {
			if trx.SenderPubkey == userSignPubkey {
				trxs = append(trxs, trx)
			}
		}
	}
	return trxs, nil
}

func dfs(blocks []*quorumpb.Block, cache map[string]bool, result []string, nodename string) error {
	for _, block := range blocks {
		if _, ok := cache[block.BlockId]; !ok {
			cache[block.BlockId] = true
			result = append(result, block.BlockId)
			subBlocks, err := nodectx.GetNodeCtx().GetChainStorage().GetSubBlock(block.BlockId, nodename)
			if err != nil {
				return err
			}
			err = dfs(subBlocks, cache, result, nodename)
		}
	}
	return nil
}
