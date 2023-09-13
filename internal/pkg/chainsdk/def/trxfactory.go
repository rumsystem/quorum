package def

import (
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type TrxFactoryIface interface {
	GetChainConfigTrx(senderPubkey, senderKeyname string, item *quorumpb.ChainConfigItem) (*quorumpb.Trx, error)
	GetUpdAppConfigTrx(senderPubkey, senderKeyname string, item *quorumpb.AppConfigItem) (*quorumpb.Trx, error)
	GetUpdGroupSyncerTrx(senderPubkey, senderKeyname string, item *quorumpb.UpdGroupSyncerItem) (*quorumpb.Trx, error)
	GetAddCellarReqTrx(senderPubkey, senderKeyname, cellarCipherKey string, item *quorumpb.AddCellarReqItem) (*quorumpb.Trx, error)
	GetPostAnyTrx(senderPubkey, senderKeyname string, content []byte, encryptto ...[]string) (*quorumpb.Trx, error)
	GetForkTrx(senderPubkey, senderKeyname string, item *quorumpb.ForkItem) (*quorumpb.Trx, error)
}

//GetReqBlocksTrx(keyalias string, groupId string, fromBlock uint64, blkReq int32) (*quorumpb.Trx, error)
//GetReqBlocksRespTrx(keyalias string, groupId string, requester string, fromBlock uint64, blkReq int32, blocks []*quorumpb.Block, result quorumpb.ReqBlkResult) (*quorumpb.Trx, error)
//GetChangeConsensusResultTrx(keyalias string, trxId string, item *quorumpb.ChangeConsensusResultBundle) (*quorumpb.Trx, error)
