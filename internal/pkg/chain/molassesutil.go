package chain

import (
	"bytes"

	localCrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

var molautil_log = logging.Logger("util")

//find the highest block from the block tree
func RecalChainHeight(blocks []*quorumpb.Block, currentHeight int64, currentHighestBlock *quorumpb.Block, nodename string) (int64, string, error) {
	molautil_log.Debug("RecalChainHeight called")

	newHighestHeight := currentHeight
	newHighestBlockId := currentHighestBlock.BlockId
	newHighestBlock := currentHighestBlock

	for _, block := range blocks {
		blockHeight, err := nodectx.GetDbMgr().GetBlockHeight(block.BlockId, nodename)
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
func GetTrimedBlocks(blocks []*quorumpb.Block, nodename string) ([]string, error) {
	molautil_log.Debug("GetTrimedBlocks called")
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

func dfs(blocks []*quorumpb.Block, cache map[string]bool, result []string, nodename string) error {
	molautil_log.Debug("dfs called")
	for _, block := range blocks {
		if _, ok := cache[block.BlockId]; !ok {
			cache[block.BlockId] = true
			result = append(result, block.BlockId)
			subBlocks, err := nodectx.GetDbMgr().GetSubBlock(block.BlockId, nodename)
			if err != nil {
				return err
			}
			err = dfs(subBlocks, cache, result, nodename)
		}
	}
	return nil
}

//get all trx belongs to me from the block list
func GetMyTrxs(blockIds []string, nodename string, userSignPubkey string) ([]*quorumpb.Trx, error) {
	molautil_log.Debug("GetMyTrxs called")
	var trxs []*quorumpb.Trx

	for _, blockId := range blockIds {
		block, err := nodectx.GetDbMgr().GetBlock(blockId, false, nodename)
		if err != nil {
			chain_log.Warnf(err.Error())
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

//get all trx from the block list
func GetAllTrxs(blocks []*quorumpb.Block) ([]*quorumpb.Trx, error) {
	molautil_log.Debug("GetAllTrxs called")
	var trxs []*quorumpb.Trx
	for _, block := range blocks {
		for _, trx := range block.Trxs {
			trxs = append(trxs, trx)
		}
	}
	return trxs, nil
}

//update resend count (+1) for all trxs
func UpdateResendCount(trxs []*quorumpb.Trx) ([]*quorumpb.Trx, error) {
	molautil_log.Debug("UpdateResendCount called")
	for _, trx := range trxs {
		trx.ResendCount++
	}
	return trxs, nil
}

func Hash(data []byte) []byte {
	return localCrypto.Hash(data)
}
