package data

import (
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type TrxFactory struct {
	nodename  string
	groupId   string
	version   string
	CipherKey string
}

func (factory *TrxFactory) Init(nodename, version, groupId, CipheryKey string) {
	factory.groupId = groupId
	factory.nodename = nodename
	factory.version = version
	factory.CipherKey = CipheryKey
}

func (factory *TrxFactory) CreateTrxByEthKey(msgType quorumpb.TrxType, data []byte, senderPubkey, senderKeyname string, encryptto ...[]string) (*quorumpb.Trx, error) {
	return CreateTrxByEthKey(factory.nodename, factory.version, factory.groupId, senderPubkey, senderKeyname, factory.CipherKey, msgType, data, encryptto...)
}

func (factory *TrxFactory) GetUpdAppConfigTrx(senderPubkey, senderKeyname string, item *quorumpb.AppConfigItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrxByEthKey(quorumpb.TrxType_APP_CONFIG, encodedcontent, senderPubkey, senderKeyname)
}

func (factory *TrxFactory) GetUpdGroupSyncerTrx(senderPubkey, senderKeyname string, item *quorumpb.UpdGroupSyncerItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}
	return factory.CreateTrxByEthKey(quorumpb.TrxType_UPD_SYNCER, encodedcontent, senderPubkey, senderKeyname)
}

func (factory *TrxFactory) GetUpdGroupPosterTrx(senderPubkey, senderKeyname string, item *quorumpb.UpdGroupPosterItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrxByEthKey(quorumpb.TrxType_UPD_POSTER, encodedcontent, senderPubkey, senderKeyname)
}

func (factory *TrxFactory) GetPostAnyTrx(senderPubkey, senderKeyname string, content []byte, encryptto ...[]string) (*quorumpb.Trx, error) {
	if _, err := IsTrxDataWithinSizeLimit(content); err != nil {
		return nil, err
	}

	return factory.CreateTrxByEthKey(quorumpb.TrxType_POST, content, senderPubkey, senderKeyname)
}

func (factory *TrxFactory) GetForkTrx(senderPubkey, senderKeyname string, item *quorumpb.ForkItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrxByEthKey(quorumpb.TrxType_FORK, encodedcontent, senderPubkey, senderKeyname)
}

func (factory *TrxFactory) GetAddCellarReqTrx(senderPubkey, senderKeyname, cellarCipherKey string, item *quorumpb.ReqGroupServiceItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}

	return CreateTrxByEthKey(factory.nodename, factory.version, factory.groupId, senderPubkey, senderKeyname, cellarCipherKey, quorumpb.TrxType_SERVICE_REQ, encodedcontent)

}
