package data

import (
	"encoding/binary"
	"errors"

	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type TrxFactory struct {
	nodename   string
	groupId    string
	groupItem  *quorumpb.GroupItem
	chainNonce ChainNonce
	version    string
}

type ChainNonce interface {
	GetNextNonce(groupId string, prefix ...string) (nonce uint64, err error)
}

func (factory *TrxFactory) Init(version string, groupItem *quorumpb.GroupItem, nodename string, chainnonce ChainNonce) {
	factory.groupItem = groupItem
	factory.groupId = groupItem.GroupId
	factory.nodename = nodename
	factory.chainNonce = chainnonce
	factory.version = version
}

func (factory *TrxFactory) CreateTrxByEthKey(msgType quorumpb.TrxType, data []byte, keyalias string, encryptto ...[]string) (*quorumpb.Trx, error) {
	nonce, err := factory.chainNonce.GetNextNonce(factory.groupItem.GroupId, factory.nodename)
	if err != nil {
		return nil, err
	}
	return CreateTrxByEthKey(factory.nodename, factory.version, factory.groupItem, msgType, int64(nonce), data, keyalias, encryptto...)
}

func (factory *TrxFactory) GetUpdAppConfigTrx(keyalias string, item *quorumpb.AppConfigItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrxByEthKey(quorumpb.TrxType_APP_CONFIG, encodedcontent, keyalias)
}

func (factory *TrxFactory) GetChainConfigTrx(keyalias string, item *quorumpb.ChainConfigItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrxByEthKey(quorumpb.TrxType_CHAIN_CONFIG, encodedcontent, keyalias)
}

func (factory *TrxFactory) GetRegProducerTrx(keyalias string, item *quorumpb.ProducerItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}
	return factory.CreateTrxByEthKey(quorumpb.TrxType_PRODUCER, encodedcontent, keyalias)
}

func (factory *TrxFactory) GetRegUserTrx(keyalias string, item *quorumpb.UserItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}
	return factory.CreateTrxByEthKey(quorumpb.TrxType_USER, encodedcontent, keyalias)
}

func (factory *TrxFactory) GetAnnounceTrx(keyalias string, item *quorumpb.AnnounceItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrxByEthKey(quorumpb.TrxType_ANNOUNCE, encodedcontent, keyalias)
}

func (factory *TrxFactory) GetUpdSchemaTrx(keyalias string, item *quorumpb.SchemaItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrxByEthKey(quorumpb.TrxType_SCHEMA, encodedcontent, keyalias)
}

func (factory *TrxFactory) GetReqBlockRespTrx(keyalias string, requester string, block *quorumpb.Block, result quorumpb.ReqBlkResult) (*quorumpb.Trx, error) {
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
	return CreateTrxByEthKey(factory.nodename, factory.version, factory.groupItem, quorumpb.TrxType_REQ_BLOCK_RESP, int64(0), bItemBytes, keyalias)
}

func (factory *TrxFactory) GetReqBlockForwardTrx(keyalias string, block *quorumpb.Block) (*quorumpb.Trx, error) {
	var reqBlockItem quorumpb.ReqBlock
	reqBlockItem.BlockId = block.BlockId
	reqBlockItem.GroupId = block.GroupId
	reqBlockItem.UserId = factory.groupItem.UserSignPubkey

	bItemBytes, err := proto.Marshal(&reqBlockItem)
	if err != nil {
		return nil, err
	}

	return CreateTrxByEthKey(factory.nodename, factory.version, factory.groupItem, quorumpb.TrxType_REQ_BLOCK_FORWARD, int64(0), bItemBytes, keyalias)
}

func (factory *TrxFactory) GetReqBlockBackwardTrx(keyalias string, block *quorumpb.Block) (*quorumpb.Trx, error) {
	var reqBlockItem quorumpb.ReqBlock
	reqBlockItem.BlockId = block.BlockId
	reqBlockItem.GroupId = block.GroupId
	reqBlockItem.UserId = factory.groupItem.UserSignPubkey

	bItemBytes, err := proto.Marshal(&reqBlockItem)
	if err != nil {
		return nil, err
	}

	return CreateTrxByEthKey(factory.nodename, factory.version, factory.groupItem, quorumpb.TrxType_REQ_BLOCK_BACKWARD, int64(0), bItemBytes, keyalias)
}

func (factory *TrxFactory) GetBlockProducedTrx(keyalias string, blk *quorumpb.Block) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(blk)
	if err != nil {
		return nil, err
	}
	return CreateTrxByEthKey(factory.nodename, factory.version, factory.groupItem, quorumpb.TrxType_BLOCK_PRODUCED, int64(0), encodedcontent, keyalias)
}

func (factory *TrxFactory) GetPostAnyTrx(keyalias string, content []byte, encryptto ...[]string) (*quorumpb.Trx, error) {
	if binary.Size(content) > OBJECT_SIZE_LIMIT {
		err := errors.New("Content size over 200Kb")
		return nil, err
	}

	return factory.CreateTrxByEthKey(quorumpb.TrxType_POST, content, keyalias, encryptto...)
}
