package def

import (
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type TrxFactoryIface interface {
	GetAnnounceTrx(keyalias string, item *quorumpb.AnnounceItem) (*quorumpb.Trx, error)
	GetChainConfigTrx(keyalias string, item *quorumpb.ChainConfigItem) (*quorumpb.Trx, error)
	GetUpdAppConfigTrx(keyalias string, item *quorumpb.AppConfigItem) (*quorumpb.Trx, error)
	GetUpdGroupUserTrx(keyalias string, item *quorumpb.UpdGroupUserItem) (*quorumpb.Trx, error)
	GetPostAnyTrx(keyalias string, content []byte, encryptto ...[]string) (*quorumpb.Trx, error)
	GetForkTrx(keyalias string, item *quorumpb.ForkItem) (*quorumpb.Trx, error)
}

//GetReqBlocksTrx(keyalias string, groupId string, fromBlock uint64, blkReq int32) (*quorumpb.Trx, error)
//GetReqBlocksRespTrx(keyalias string, groupId string, requester string, fromBlock uint64, blkReq int32, blocks []*quorumpb.Block, result quorumpb.ReqBlkResult) (*quorumpb.Trx, error)
//GetChangeConsensusResultTrx(keyalias string, trxId string, item *quorumpb.ChangeConsensusResultBundle) (*quorumpb.Trx, error)
