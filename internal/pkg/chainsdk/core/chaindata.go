package chain

import (
	"encoding/hex"
	"errors"
	"fmt"

	guuid "github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"

	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type ChainData struct {
	nodename       string
	groupId        string
	groupCipherKey string
	userSignPubkey string
	dbmgr          *storage.DbMgr
}

func (d *ChainData) GetBlockForwardByReqTrx(trx *quorumpb.Trx, cipherKey string, prefix ...string) ([]*quorumpb.Block, error) {
	chain_log.Debugf("<%s> GetBlockForward called", trx.GroupId)
	var reqBlockItem quorumpb.ReqBlock
	bytecipherKey, err := hex.DecodeString(cipherKey)
	if err != nil {
		return nil, err
	}

	decryptData, err := localcrypto.AesDecode(trx.Data, bytecipherKey)
	if err != nil {
		return nil, err
	}

	if err := proto.Unmarshal(decryptData, &reqBlockItem); err != nil {
		return nil, err
	}

	//commented by cuicat
	/*
		//check if requester is in group block list
		isBlocked, _ := d.dbmgr.IsUserBlocked(trx.GroupId, trx.SenderPubkey, prefix...)
		if isBlocked {
			molaproducer_log.Debugf("<%s> user <%s> is blocked", trx.GroupId, trx.SenderPubkey)
			return nil, nil
		}
	*/

	//added by cuicat
	//check if trx sender is in group block list
	isAllow, err := nodectx.GetNodeCtx().GetChainStorage().CheckTrxTypeAuth(trx.GroupId, trx.SenderPubkey, trx.Type, prefix...)
	if err != nil {
		return nil, nil
	}

	if !isAllow {
		chain_log.Debugf("<%s> user <%s> don't has permission on trx type <%s>", trx.GroupId, trx.SenderPubkey, trx.Type.String())
		return nil, nil
	}

	chain_log.Debugf("<%s> GetBlockForward block id: %s", trx.GroupId, reqBlockItem.BlockId)

	subBlocks, err := nodectx.GetNodeCtx().GetChainStorage().GetSubBlock(reqBlockItem.BlockId, prefix...)
	return subBlocks, err
}

func (d *ChainData) GetBlockBackwardByReqTrx(trx *quorumpb.Trx, cipherKey string, prefix ...string) (string, *quorumpb.Block, error) {
	chain_log.Debugf("<%s> GetBlockBackwardcalled", trx.GroupId)

	var reqBlockItem quorumpb.ReqBlock
	bytecipherKey, err := hex.DecodeString(cipherKey)
	if err != nil {
		return "", nil, err
	}

	decryptData, err := localcrypto.AesDecode(trx.Data, bytecipherKey)
	if err != nil {
		return "", nil, err
	}

	if err := proto.Unmarshal(decryptData, &reqBlockItem); err != nil {
		return "", nil, err
	}

	//commented by cuicat
	/*
		//check if requester is in group block list
		isBlocked, _ := d.dbmgr.IsUserBlocked(trx.GroupId, trx.SenderPubkey, prefix...)
		if isBlocked {
			molaproducer_log.Debugf("<%s> user <%s> is blocked", trx.GroupId, trx.SenderPubkey)
			return nil, nil
		}
	*/

	//added by cuicat
	//check if trx sender is in group block list
	isAllow, err := nodectx.GetNodeCtx().GetChainStorage().CheckTrxTypeAuth(trx.GroupId, trx.SenderPubkey, trx.Type, prefix...)
	if err != nil {
		return reqBlockItem.BlockId, nil, nil
	}

	if !isAllow {
		chain_log.Debugf("<%s> user <%s> don't has permission on trx type <%s>", trx.GroupId, trx.SenderPubkey, trx.Type.String())
		return reqBlockItem.BlockId, nil, nil
	}

	isExist, err := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(reqBlockItem.BlockId, false, prefix...)
	if err != nil {
		return reqBlockItem.BlockId, nil, err
	} else if !isExist {
		return reqBlockItem.BlockId, nil, errors.New("Block not exist")
	}

	block, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(reqBlockItem.BlockId, false, prefix...)
	if err != nil {
		return reqBlockItem.BlockId, nil, err
	}

	isParentExit, err := nodectx.GetNodeCtx().GetChainStorage().IsParentExist(block.PrevBlockId, false, prefix...)
	if err != nil {
		return reqBlockItem.BlockId, nil, err
	}
	if isParentExit {
		parentBlock, err := nodectx.GetNodeCtx().GetChainStorage().GetParentBlock(reqBlockItem.BlockId, prefix...)
		return reqBlockItem.BlockId, parentBlock, err
	}
	return reqBlockItem.BlockId, nil, errors.New("Parent Block not exist")
}

func (d *ChainData) CreateReqBlockResp(cipherKey string, trx *quorumpb.Trx, block *quorumpb.Block, userSignPubkey string, result quorumpb.ReqBlkResult) (*quorumpb.ReqBlockResp, error) {

	var reqBlockItem quorumpb.ReqBlock
	ciperKey, err := hex.DecodeString(cipherKey)
	if err != nil {
		return nil, err
	}

	decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
	if err != nil {
		return nil, err
	}

	if err := proto.Unmarshal(decryptData, &reqBlockItem); err != nil {
		return nil, err
	}

	var reqBlockRespItem quorumpb.ReqBlockResp
	reqBlockRespItem.Result = result
	reqBlockRespItem.ProviderPubkey = userSignPubkey
	reqBlockRespItem.RequesterPubkey = reqBlockItem.UserId
	reqBlockRespItem.GroupId = reqBlockItem.GroupId
	reqBlockRespItem.BlockId = reqBlockItem.BlockId

	pbBytesBlock, err := proto.Marshal(block)
	if err != nil {
		return nil, err
	}
	reqBlockRespItem.Block = pbBytesBlock
	return &reqBlockRespItem, nil
}

func (d *ChainData) GetBlockForward(trx *quorumpb.Trx) (requester string, blocks []*quorumpb.Block, isEmptyBlock bool, erer error) {
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

	var subBlocks []*quorumpb.Block
	subBlocks, err = nodectx.GetNodeCtx().GetChainStorage().GetSubBlock(reqBlockItem.BlockId, d.nodename)
	if err != nil {
		return "", nil, false, err
	}

	if len(subBlocks) != 0 {
		return reqBlockItem.UserId, subBlocks, false, nil
	} else {
		var emptyBlock *quorumpb.Block
		emptyBlock = &quorumpb.Block{}
		emptyBlock.PrevBlockId = reqBlockItem.BlockId
		emptyBlock.ProducerPubKey = d.userSignPubkey
		subBlocks = append(subBlocks, emptyBlock)
		return reqBlockItem.UserId, subBlocks, true, nil
	}
}

func (d *ChainData) GetBlockBackward(trx *quorumpb.Trx) (requester string, block *quorumpb.Block, isEmptyBlock bool, err error) {
	chain_log.Debugf("<%s> GetBlockBackward called", d.groupId)

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

	//check previllage
	isAllow, err := nodectx.GetNodeCtx().GetChainStorage().CheckTrxTypeAuth(trx.GroupId, trx.SenderPubkey, quorumpb.TrxType_REQ_BLOCK_BACKWARD, d.nodename)
	if err != nil {
		return "", nil, false, err
	}

	if !isAllow {
		chain_log.Debugf("<%s> user <%s>: trxType <%s> is denied", d.groupId, trx.SenderPubkey, quorumpb.TrxType_REQ_BLOCK_BACKWARD.String())
		return reqBlockItem.UserId, nil, false, errors.New("insufficient privileges")
	}

	isExist, err := nodectx.GetNodeCtx().GetChainStorage().IsBlockExist(reqBlockItem.BlockId, false, d.nodename)
	if err != nil {
		return "", nil, false, err
	} else if !isExist {
		return "", nil, false, fmt.Errorf("Block not exist")
	}

	blk, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(reqBlockItem.BlockId, false, d.nodename)
	if err != nil {
		return "", nil, false, err
	}

	isParentExit, err := nodectx.GetNodeCtx().GetChainStorage().IsParentExist(blk.PrevBlockId, false, d.nodename)
	if err != nil {
		return "", nil, false, err
	}

	if isParentExit {
		chain_log.Debugf("<%s> send REQ_NEXT_BLOCK_RESP (BLOCK_IN_TRX)", d.groupId)
		parentBlock, err := nodectx.GetNodeCtx().GetChainStorage().GetParentBlock(reqBlockItem.BlockId, d.nodename)
		if err != nil {
			return "", nil, false, err
		}

		return reqBlockItem.UserId, parentBlock, false, nil
	} else {
		chain_log.Debugf("<%s> send REQ_NEXT_BLOCK_RESP (BLOCK_NOT_FOUND)", d.groupId)
		var emptyBlock *quorumpb.Block
		emptyBlock = &quorumpb.Block{}
		emptyBlock.BlockId = guuid.New().String()
		emptyBlock.ProducerPubKey = d.userSignPubkey
		return reqBlockItem.UserId, emptyBlock, true, nil
	}
}
