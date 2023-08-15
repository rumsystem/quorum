package data

import (
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type TrxFactory struct {
	nodename  string
	groupId   string
	groupItem *quorumpb.GroupItem
	version   string
}

func (factory *TrxFactory) Init(version string, groupItem *quorumpb.GroupItem, nodename string) {
	factory.groupItem = groupItem
	factory.groupId = groupItem.GroupId
	factory.nodename = nodename
	factory.version = version
}

func (factory *TrxFactory) CreateTrxByEthKey(msgType quorumpb.TrxType, data []byte, keyalias string, encryptto ...[]string) (*quorumpb.Trx, error) {
	return CreateTrxByEthKey(factory.nodename, factory.version, factory.groupItem, msgType, data, keyalias, encryptto...)
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

func (factory *TrxFactory) GetUpdGroupUserTrx(keyalias string, item *quorumpb.UpdGroupUserItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}
	return factory.CreateTrxByEthKey(quorumpb.TrxType_UPD_GRP_USER, encodedcontent, keyalias)
}

func (factory *TrxFactory) GetPostAnyTrx(keyalias string, content []byte, encryptto ...[]string) (*quorumpb.Trx, error) {
	if _, err := IsTrxDataWithinSizeLimit(content); err != nil {
		return nil, err
	}

	return factory.CreateTrxByEthKey(quorumpb.TrxType_POST, content, keyalias, encryptto...)
}

func (factory *TrxFactory) GetForkTrx(keyalis string, item *quorumpb.ForkItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrxByEthKey(quorumpb.TrxType_FORK, encodedcontent, keyalis)
}
