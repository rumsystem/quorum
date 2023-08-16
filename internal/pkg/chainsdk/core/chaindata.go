package chain

import (
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	"google.golang.org/protobuf/proto"

	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type ChainData struct {
	nodename       string
	groupId        string
	groupCipherKey string
	userSignPubkey string
	dbmgr          *storage.DbMgr
}

// TBD, move this to chain confi
const MAX_BLOCK_IN_RESP_BYTES = 10485760 //10MB

func (d *ChainData) GetReqBlocks(req *quorumpb.ReqBlock) (blocks []*quorumpb.Block, result quorumpb.ReqBlkResult, err error) {
	chain_log.Debugf("<%s> GetReqBlocks called", d.groupId)

	exist := false
	exist, err = nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(req.GroupId, req.FromBlock, false, d.nodename)
	if err != nil {
		return nil, -1, err
	}

	if !exist {
		return nil, quorumpb.ReqBlkResult_BLOCK_NOT_FOUND, nil
	}

	var bs []*quorumpb.Block
	currBlock := req.FromBlock
	totalBlockBytes := 0
	totalBlockPackaged := 0

	for totalBlockPackaged < int(req.BlksRequested) {
		block, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(req.GroupId, currBlock, false, d.nodename)
		if err != nil {
			return nil, -1, err
		}
		//get data length
		pdate, _ := proto.Marshal(block)
		totalBlockBytes = totalBlockBytes + len(pdate)
		//check if reach maximum length, may have more
		if totalBlockBytes > MAX_BLOCK_IN_RESP_BYTES {
			return blocks, quorumpb.ReqBlkResult_BLOCK_IN_RESP, nil
		}

		//put block into blocks list
		bs = append(bs, block)
		totalBlockPackaged += 1
		currBlock += 1
		//check if next epoch exist
		exist, err = nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(req.GroupId, currBlock, false, d.nodename)
		if err != nil {
			return nil, -1, err
		}

		//no more and on top
		if !exist {
			return bs, quorumpb.ReqBlkResult_BLOCK_IN_RESP_ON_TOP, nil
		}
		//continue and put more
	}

	return bs, quorumpb.ReqBlkResult_BLOCK_IN_RESP, nil
}
