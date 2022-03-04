package chain

import (
	"encoding/hex"
	"errors"

	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	"google.golang.org/protobuf/proto"
)

type ChainData struct {
	dbmgr *storage.DbMgr
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
	isAllow, err := d.dbmgr.CheckTrxTypeAuth(trx.GroupId, trx.SenderPubkey, trx.Type, prefix...)
	if err != nil {
		return nil, nil
	}

	if !isAllow {
		chain_log.Debugf("<%s> user <%s> don't has permission on trx type <%s>", trx.GroupId, trx.SenderPubkey, trx.Type.String())
		return nil, nil
	}

	chain_log.Debugf("<%s> GetBlockForward block id: %s", trx.GroupId, reqBlockItem.BlockId)

	subBlocks, err := d.dbmgr.GetSubBlock(reqBlockItem.BlockId, prefix...)
	return subBlocks, err
}

func (d *ChainData) GetBlockBackwardByReqTrx(trx *quorumpb.Trx, cipherKey string, prefix ...string) (*quorumpb.Block, error) {
	chain_log.Debugf("<%s> GetBlockBackwardcalled", trx.GroupId)

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
	isAllow, err := d.dbmgr.CheckTrxTypeAuth(trx.GroupId, trx.SenderPubkey, trx.Type, prefix...)
	if err != nil {
		return nil, nil
	}

	if !isAllow {
		chain_log.Debugf("<%s> user <%s> don't has permission on trx type <%s>", trx.GroupId, trx.SenderPubkey, trx.Type.String())
		return nil, nil
	}

	isExist, err := d.dbmgr.IsBlockExist(reqBlockItem.BlockId, false, prefix...)
	if err != nil {
		return nil, err
	} else if !isExist {
		return nil, errors.New("Block not exist")
	}

	block, err := d.dbmgr.GetBlock(reqBlockItem.BlockId, false, prefix...)
	if err != nil {
		return nil, err
	}

	isParentExit, err := d.dbmgr.IsParentExist(block.PrevBlockId, false, prefix...)
	if err != nil {
		return nil, err
	}
	if isParentExit {
		parentBlock, err := d.dbmgr.GetParentBlock(reqBlockItem.BlockId, prefix...)
		return parentBlock, err
	}
	return nil, errors.New("Parent Block not exist")
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
