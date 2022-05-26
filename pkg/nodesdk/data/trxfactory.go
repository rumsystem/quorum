package data

import (
	"encoding/binary"
	"errors"

	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type TrxFactory struct {
	groupId          string
	nodesdkGroupItem *quorumpb.NodeSDKGroupItem
	chainNonce       ChainNonce
	version          string
}

type ChainNonce interface {
	GetNextNouce(groupId string, prefix ...string) (nonce uint64, err error)
}

func (factory *TrxFactory) Init(version string, nodesdkGroupItem *quorumpb.NodeSDKGroupItem, chainnonce ChainNonce) {
	factory.nodesdkGroupItem = nodesdkGroupItem
	factory.groupId = nodesdkGroupItem.Group.GroupId
	factory.chainNonce = chainnonce
	factory.version = version
}

func (factory *TrxFactory) CreateTrx(msgType quorumpb.TrxType, data []byte, encryptto ...[]string) (*quorumpb.Trx, error) {
	nonce, err := factory.chainNonce.GetNextNouce(factory.nodesdkGroupItem.Group.GroupId)
	if err != nil {
		return nil, err
	}
	return CreateTrx(factory.version, factory.nodesdkGroupItem, msgType, int64(nonce), data)
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
