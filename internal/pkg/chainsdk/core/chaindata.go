package chain

import (
	"errors"

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

func (d *ChainData) GetReqBlocks(req *quorumpb.ReqBlock) (requester string, fromBlock uint64, reqBlocks int32, blocks []*quorumpb.Block, result quorumpb.ReqBlkResult, err error) {
	chain_log.Debugf("<%s> GetReqBlocks called", d.groupId)

	isAllow, err := nodectx.GetNodeCtx().GetChainStorage().CheckPackageTypeAuth(req.GroupId, req.ReqPubkey, quorumpb.PackageType_SYNC_MSG, d.nodename)
	if err != nil {
		return "", 0, 0, nil, -1, err
	}

	if !isAllow {
		chain_log.Debugf("<%s> pubkey <%s>: package type <%s> is not allowed", d.groupId, req.ReqPubkey, quorumpb.PackageType_SYNC_MSG.String())
		return "", 0, 0, nil, -1, errors.New("requester don't have sufficient privileges")
	}

	exist := false
	exist, err = nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(req.GroupId, req.FromBlock, false, d.nodename)
	if err != nil {
		return "", 0, 0, nil, -1, err
	}

	if !exist {
		return req.ReqPubkey, req.FromBlock, req.BlksRequested, nil, quorumpb.ReqBlkResult_BLOCK_NOT_FOUND, nil
	}

	var bs []*quorumpb.Block
	currBlock := req.FromBlock
	totalBlockBytes := 0
	totalBlockPackaged := 0

	for totalBlockPackaged < int(req.BlksRequested) {
		block, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(req.GroupId, currBlock, false, d.nodename)
		if err != nil {
			return "", 0, 0, nil, -1, err
		}
		//get data length
		pdate, _ := proto.Marshal(block)
		totalBlockBytes = totalBlockBytes + len(pdate)
		//check if reach maximum length, may have more
		if totalBlockBytes > MAX_BLOCK_IN_RESP_BYTES {
			return req.ReqPubkey, req.FromBlock, req.BlksRequested, blocks, quorumpb.ReqBlkResult_BLOCK_IN_RESP, nil
		}

		//put block into blocks list
		bs = append(bs, block)
		totalBlockPackaged += 1
		currBlock += 1
		//check if next epoch exist
		exist, err = nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(req.GroupId, currBlock, false, d.nodename)
		if err != nil {
			return "", 0, 0, nil, -1, err
		}

		//no more and on top
		if !exist {
			return req.ReqPubkey, req.FromBlock, req.BlksRequested, bs, quorumpb.ReqBlkResult_BLOCK_IN_RESP_ON_TOP, nil
		}
		//continue and put more
	}

	return req.ReqPubkey, req.FromBlock, req.BlksRequested, bs, quorumpb.ReqBlkResult_BLOCK_IN_RESP, nil
}
