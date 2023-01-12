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
	GetNextNouce(groupId string, prefix ...string) (nonce uint64, err error)
}

func (factory *TrxFactory) Init(version string, groupItem *quorumpb.GroupItem, nodename string, chainnonce ChainNonce) {
	factory.groupItem = groupItem
	factory.groupId = groupItem.GroupId
	factory.nodename = nodename
	factory.chainNonce = chainnonce
	factory.version = version
}

func (factory *TrxFactory) CreateTrxByEthKey(msgType quorumpb.TrxType, data []byte, keyalias string, encryptto ...[]string) (*quorumpb.Trx, error) {
	nonce, err := factory.chainNonce.GetNextNouce(factory.groupItem.GroupId, factory.nodename)
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

func (factory *TrxFactory) GetRegProducerBundleTrx(keyalias string, item *quorumpb.BFTProducerBundleItem) (*quorumpb.Trx, error) {
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

func (factory *TrxFactory) GetReqBlocksRespTrx(keyalias string, groupId string, requester string, blkReq int64, fromEpoch int64, blocks []*quorumpb.Block, result quorumpb.ReqBlkResult) (*quorumpb.Trx, error) {
	var reqBlockRespItem quorumpb.ReqBlockResp
	reqBlockRespItem.GroupId = groupId
	reqBlockRespItem.Result = result
	reqBlockRespItem.RequesterPubkey = requester
	reqBlockRespItem.ProviderPubkey = factory.groupItem.UserSignPubkey
	reqBlockRespItem.FromEpoch = fromEpoch
	reqBlockRespItem.BlksRequested = int64(blkReq)
	reqBlockRespItem.BlksProvided = int64(len(blocks))
	blockBundles := &quorumpb.BlocksBundle{}
	blockBundles.Blocks = blocks
	reqBlockRespItem.Blocks = blockBundles

	bItemBytes, err := proto.Marshal(&reqBlockRespItem)
	if err != nil {
		return nil, err
	}

	//send ask next block trx out
	return CreateTrxByEthKey(factory.nodename, factory.version, factory.groupItem, quorumpb.TrxType_REQ_BLOCK_RESP, int64(0), bItemBytes, keyalias)
}

func (factory *TrxFactory) GetReqBlocksTrx(keyalias string, groupId string, fromEpoch int64, blkReq int64) (*quorumpb.Trx, error) {
	var reqBlockItem quorumpb.ReqBlock
	reqBlockItem.GroupId = groupId
	reqBlockItem.FromEpoch = int64(fromEpoch)
	reqBlockItem.BlksRequested = int64(blkReq)
	reqBlockItem.ReqPubkey = factory.groupItem.UserSignPubkey

	bItemBytes, err := proto.Marshal(&reqBlockItem)
	if err != nil {
		return nil, err
	}

	return CreateTrxByEthKey(factory.nodename, factory.version, factory.groupItem, quorumpb.TrxType_REQ_BLOCK, int64(0), bItemBytes, keyalias)
}

func (factory *TrxFactory) GetPostAnyTrx(keyalias string, content proto.Message, encryptto ...[]string) (*quorumpb.Trx, error) {
	encodedcontent, err := quorumpb.ContentToBytes(content)
	if err != nil {
		return nil, err
	}

	if binary.Size(encodedcontent) > OBJECT_SIZE_LIMIT {
		err := errors.New("content size over 0.9MB")
		return nil, err
	}

	return factory.CreateTrxByEthKey(quorumpb.TrxType_POST, encodedcontent, keyalias, encryptto...)
}
