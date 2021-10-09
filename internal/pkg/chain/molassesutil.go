package chain

import (
	localCrypto "github.com/huo-ju/quorum/internal/pkg/crypto"
	"github.com/huo-ju/quorum/internal/pkg/nodectx"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	logging "github.com/ipfs/go-log/v2"
)

var molautil_log = logging.Logger("util")

//find the highest block from the block tree
func RecalChainHeight(blocks []*quorumpb.Block, currentHeight int64, nodename string) (int64, []string, error) {
	molautil_log.Debug("RecalChainHeight called")
	var highestBlockId []string
	newHeight := currentHeight
	for _, block := range blocks {
		blockHeight, err := nodectx.GetDbMgr().GetBlockHeight(block.BlockId, nodename)
		if err != nil {
			return -1, highestBlockId, err
		}
		if blockHeight > newHeight {
			newHeight = blockHeight
			highestBlockId = nil
			highestBlockId = append(highestBlockId, block.BlockId)
		} else if blockHeight == newHeight {
			highestBlockId = append(highestBlockId, block.BlockId)
		} else {
			// do nothing
		}
	}
	return newHeight, highestBlockId, nil
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
