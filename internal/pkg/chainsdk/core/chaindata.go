package chain

import (
	"encoding/hex"
	"errors"

	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
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

func (d *ChainData) GetReqBlocks(trx *quorumpb.Trx) (requester string, fromBlock uint64, reqBlocks int32, blocks []*quorumpb.Block, result quorumpb.ReqBlkResult, err error) {
	chain_log.Debugf("<%s> GetReqBlocks called", d.groupId)

	var reqBlockItem quorumpb.ReqBlock
	ciperKey, err := hex.DecodeString(d.groupCipherKey)
	if err != nil {
		return "", 0, 0, nil, -1, err
	}

	decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
	if err != nil {
		return "", 0, 0, nil, -1, err
	}

	if err := proto.Unmarshal(decryptData, &reqBlockItem); err != nil {
		return "", 0, 0, nil, -1, err
	}

	//check trx sender should be same as requester in reqBlock Item
	if trx.SenderPubkey != reqBlockItem.ReqPubkey {
		return "", 0, 0, nil, -1, errors.New("trx sender/block requester mismatch")
	}

	isAllow, err := nodectx.GetNodeCtx().GetChainStorage().CheckTrxTypeAuth(trx.GroupId, trx.SenderPubkey, quorumpb.TrxType_REQ_BLOCK, d.nodename)
	if err != nil {
		return "", 0, 0, nil, -1, err
	}

	if !isAllow {
		chain_log.Debugf("<%s> user <%s>: trxType <%s> is denied", d.groupId, trx.SenderPubkey, quorumpb.TrxType_REQ_BLOCK.String())
		return "", 0, 0, nil, -1, errors.New("requester don't have sufficient privileges")
	}

	exist := false
	exist, err = nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(reqBlockItem.GroupId, reqBlockItem.FromBlock, false, d.nodename)
	if err != nil {
		return "", 0, 0, nil, -1, err
	}

	if !exist {
		return reqBlockItem.ReqPubkey, reqBlockItem.FromBlock, reqBlockItem.BlksRequested, nil, quorumpb.ReqBlkResult_BLOCK_NOT_FOUND, nil
	}

	var bs []*quorumpb.Block
	currBlock := reqBlockItem.FromBlock
	totalBlockBytes := 0
	totalBlockPackaged := 0

	for totalBlockPackaged < int(reqBlockItem.BlksRequested) {
		block, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(reqBlockItem.GroupId, currBlock, false, d.nodename)
		if err != nil {
			return "", 0, 0, nil, -1, err
		}
		//get data length
		pdate, _ := proto.Marshal(block)
		totalBlockBytes = totalBlockBytes + len(pdate)
		//check if reach maximum length, may have more
		if totalBlockBytes > MAX_BLOCK_IN_RESP_BYTES {
			return reqBlockItem.ReqPubkey, reqBlockItem.FromBlock, reqBlockItem.BlksRequested, blocks, quorumpb.ReqBlkResult_BLOCK_IN_RESP, nil
		}

		//put block into blocks list
		bs = append(bs, block)
		totalBlockPackaged += 1
		currBlock += 1
		//check if next epoch exist
		exist, err = nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(reqBlockItem.GroupId, currBlock, false, d.nodename)
		if err != nil {
			return "", 0, 0, nil, -1, err
		}

		//no more and on top
		if !exist {
			return reqBlockItem.ReqPubkey, reqBlockItem.FromBlock, reqBlockItem.BlksRequested, bs, quorumpb.ReqBlkResult_BLOCK_IN_RESP_ON_TOP, nil
		}
		//continue and put more
	}

	return reqBlockItem.ReqPubkey, reqBlockItem.FromBlock, reqBlockItem.BlksRequested, bs, quorumpb.ReqBlkResult_BLOCK_IN_RESP, nil
}
