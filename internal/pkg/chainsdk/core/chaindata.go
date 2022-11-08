package chain

import (
	"encoding/hex"
	"errors"

	"github.com/rumsystem/quorum/internal/pkg/nodectx"

	"github.com/rumsystem/quorum/internal/pkg/storage"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type ChainData struct {
	nodename       string
	groupId        string
	groupCipherKey string
	userSignPubkey string
	dbmgr          *storage.DbMgr
}

func (d *ChainData) GetBlockForward(trx *quorumpb.Trx) (requester string, block *quorumpb.Block, isEmptyBlock bool, erer error) {
	chain_log.Debugf("<%s> GetBlockForward called", d.groupId)

	var reqBlockItem quorumpb.ReqBlock
	ciperKey, err := hex.DecodeString(d.groupCipherKey)
	if err != nil {
		return "", nil, false, err
	}

	decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
	if err != nil {
		return "", nil, false, err
	}

	if err := proto.Unmarshal(decryptData, &reqBlockItem); err != nil {
		return "", nil, false, err
	}

	isAllow, err := nodectx.GetNodeCtx().GetChainStorage().CheckTrxTypeAuth(trx.GroupId, trx.SenderPubkey, quorumpb.TrxType_REQ_BLOCK_FORWARD, d.nodename)
	if err != nil {
		return "", nil, false, err
	}

	if !isAllow {
		chain_log.Debugf("<%s> user <%s>: trxType <%s> is denied", d.groupId, trx.SenderPubkey, quorumpb.TrxType_REQ_BLOCK_FORWARD.String())
		return reqBlockItem.UserId, nil, false, errors.New("insufficient privileges")
	}

	exist := false
	exist, err = nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(trx.GroupId, reqBlockItem.Epoch, false, d.nodename)
	if err != nil {
		return "", nil, false, err
	}
	if exist == false {
		var emptyBlock *quorumpb.Block
		emptyBlock = &quorumpb.Block{}
		emptyBlock.Epoch = reqBlockItem.Epoch
		return reqBlockItem.UserId, emptyBlock, true, nil
	}

	block, err = nodectx.GetNodeCtx().GetChainStorage().GetBlock(trx.GroupId, reqBlockItem.Epoch, false, d.nodename)
	if err == nil {
		return reqBlockItem.UserId, block, false, err
	} else {
		return "", nil, false, err
	}
}
