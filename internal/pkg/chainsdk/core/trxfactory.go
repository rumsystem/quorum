package chain

import (
	"encoding/binary"
	"errors"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	rumchaindata "github.com/rumsystem/rumchaindata/pkg/data"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
	"time"
)

const (
	Hours = 0
	Mins  = 0
	Sec   = 30
)

const OBJECT_SIZE_LIMIT = 200 * 1024 //(200Kb)

type TrxFactory struct {
	nodename   string
	groupId    string
	groupItem  *quorumpb.GroupItem
	chainNonce ChainNonce
}

type ChainNonce interface {
	GetNextNouce(groupId string, prefix ...string) (nonce uint64, err error)
}

func (factory *TrxFactory) Init(groupItem *quorumpb.GroupItem, nodename string, chainnonce ChainNonce) {
	factory.groupItem = groupItem
	factory.groupId = groupItem.GroupId
	factory.nodename = nodename
	factory.chainNonce = chainnonce
}

// set TimeStamp and Expired for trx
func updateTrxTimeLimit(trx *quorumpb.Trx) {
	trx.TimeStamp = time.Now().UnixNano()
	timein := time.Now().Local().Add(time.Hour*time.Duration(Hours) +
		time.Minute*time.Duration(Mins) +
		time.Second*time.Duration(Sec))
	trx.Expired = timein.UnixNano()
}

func (factory *TrxFactory) CreateTrx(msgType quorumpb.TrxType, data []byte, encryptto ...[]string) (*quorumpb.Trx, error) {
	nonce, err := factory.chainNonce.GetNextNouce(factory.groupItem.GroupId, factory.nodename)
	if err != nil {
		return nil, err
	}
	return rumchaindata.CreateTrx(factory.nodename, nodectx.GetNodeCtx().Version, factory.groupItem, msgType, int64(nonce), data, encryptto...)
}

func (factory *TrxFactory) GetUpdAppConfigTrx(item *quorumpb.AppConfigItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrx(quorumpb.TrxType_APP_CONFIG, encodedcontent)
}

func (factory *TrxFactory) GetChainConfigTrx(item *quorumpb.ChainConfigItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrx(quorumpb.TrxType_CHAIN_CONFIG, encodedcontent)
}

func (factory *TrxFactory) GetRegProducerTrx(item *quorumpb.ProducerItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}
	return factory.CreateTrx(quorumpb.TrxType_PRODUCER, encodedcontent)
}

func (factory *TrxFactory) GetRegUserTrx(item *quorumpb.UserItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}
	return factory.CreateTrx(quorumpb.TrxType_USER, encodedcontent)
}

func (factory *TrxFactory) GetAnnounceTrx(item *quorumpb.AnnounceItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrx(quorumpb.TrxType_ANNOUNCE, encodedcontent)
}

func (factory *TrxFactory) GetUpdSchemaTrx(item *quorumpb.SchemaItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrx(quorumpb.TrxType_SCHEMA, encodedcontent)
}

func (factory *TrxFactory) GetReqBlockRespTrx(requester string, block *quorumpb.Block, result quorumpb.ReqBlkResult) (*quorumpb.Trx, error) {
	var reqBlockRespItem quorumpb.ReqBlockResp
	reqBlockRespItem.Result = result
	reqBlockRespItem.ProviderPubkey = factory.groupItem.UserSignPubkey
	reqBlockRespItem.RequesterPubkey = requester
	reqBlockRespItem.GroupId = block.GroupId
	reqBlockRespItem.BlockId = block.BlockId

	pbBytesBlock, err := proto.Marshal(block)
	if err != nil {
		return nil, err
	}
	reqBlockRespItem.Block = pbBytesBlock

	bItemBytes, err := proto.Marshal(&reqBlockRespItem)
	if err != nil {
		return nil, err
	}

	//send ask next block trx out
	return factory.CreateTrx(quorumpb.TrxType_REQ_BLOCK_RESP, bItemBytes)
}

func (factory *TrxFactory) GetAskPeerIdTrx(req *quorumpb.AskPeerId) (*quorumpb.Trx, error) {
	bItemBytes, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrx(quorumpb.TrxType_ASK_PEERID, bItemBytes)
}

func (factory *TrxFactory) GetAskPeerIdRespTrx(req *quorumpb.AskPeerIdResp) (*quorumpb.Trx, error) {
	bItemBytes, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrx(quorumpb.TrxType_ASK_PEERID_RESP, bItemBytes)
}

func (factory *TrxFactory) GetReqBlockForwardTrx(block *quorumpb.Block) (*quorumpb.Trx, error) {
	var reqBlockItem quorumpb.ReqBlock
	reqBlockItem.BlockId = block.BlockId
	reqBlockItem.GroupId = block.GroupId
	reqBlockItem.UserId = factory.groupItem.UserSignPubkey

	bItemBytes, err := proto.Marshal(&reqBlockItem)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrx(quorumpb.TrxType_REQ_BLOCK_FORWARD, bItemBytes)
}

func (factory *TrxFactory) GetReqBlockBackwardTrx(block *quorumpb.Block) (*quorumpb.Trx, error) {
	var reqBlockItem quorumpb.ReqBlock
	reqBlockItem.BlockId = block.BlockId
	reqBlockItem.GroupId = block.GroupId
	reqBlockItem.UserId = factory.groupItem.UserSignPubkey

	bItemBytes, err := proto.Marshal(&reqBlockItem)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrx(quorumpb.TrxType_REQ_BLOCK_BACKWARD, bItemBytes)
}

func (factory *TrxFactory) GetBlockProducedTrx(blk *quorumpb.Block) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(blk)
	if err != nil {
		return nil, err
	}
	return factory.CreateTrx(quorumpb.TrxType_BLOCK_PRODUCED, encodedcontent)
}

func (factory *TrxFactory) GetPostAnyTrx(content proto.Message, encryptto ...[]string) (*quorumpb.Trx, error) {
	encodedcontent, err := quorumpb.ContentToBytes(content)
	if err != nil {
		return nil, err
	}

	if binary.Size(encodedcontent) > OBJECT_SIZE_LIMIT {
		err := errors.New("Content size over 200Kb")
		return nil, err
	}

	return factory.CreateTrx(quorumpb.TrxType_POST, encodedcontent, encryptto...)
}
