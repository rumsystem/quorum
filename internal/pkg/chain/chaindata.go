package chain

import (
	"encoding/hex"
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

	//check if requester is in group block list
	isBlocked, _ := d.dbmgr.IsUserBlocked(trx.GroupId, trx.SenderPubkey)
	if isBlocked {
		molaproducer_log.Debugf("<%s> user <%s> is blocked", trx.GroupId, trx.SenderPubkey)
		return nil, nil
	}

	subBlocks, err := d.dbmgr.GetSubBlock(reqBlockItem.BlockId, prefix...)
	return subBlocks, err

	//
	//	channelId := SYNC_CHANNEL_PREFIX + producer.grpItem.GroupId + "_" + reqBlockItem.UserId
	//	trxMgr, _ := producer.getSyncConn(channelId)
	//
	//	if len(subBlocks) != 0 {
	//		for _, block := range subBlocks {
	//			molaproducer_log.Debugf("<%s> send REQ_NEXT_BLOCK_RESP (BLOCK_IN_TRX)", producer.groupId)
	//			err := trxMgr.SendReqBlockResp(&reqBlockItem, block, quorumpb.ReqBlkResult_BLOCK_IN_TRX)
	//			if err != nil {
	//				molaproducer_log.Warnf(err.Error())
	//			}
	//		}
	//		return nil
	//	} else {
	//		var emptyBlock *quorumpb.Block
	//		emptyBlock = &quorumpb.Block{}
	//		//set producer pubkey of empty block
	//		emptyBlock.BlockId = guuid.New().String()
	//		emptyBlock.ProducerPubKey = producer.grpItem.UserSignPubkey
	//		molaproducer_log.Debugf("<%s> send REQ_NEXT_BLOCK_RESP (BLOCK_NOT_FOUND)", producer.groupId)
	//		return trxMgr.SendReqBlockResp(&reqBlockItem, emptyBlock, quorumpb.ReqBlkResult_BLOCK_NOT_FOUND)
	//	}
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
