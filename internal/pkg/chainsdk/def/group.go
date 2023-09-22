package def

import (
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

const LOCAL_GROUP = "local_grp"

type GroupIface interface {
	SendRawTrx(trx *quorumpb.Trx) (string, error)
	GetTrx(trxId string) (*quorumpb.Trx, bool, error)
	GetRexSyncerStatus() string
	GetCurrentBlockId() uint64
	GetBlock(blockId uint64) (*quorumpb.Block, bool, error)
	GetGroupId() string
	GetCipherKey() string
	GetOwnerPubkey() string
	GetGroupName() string
	GetConsensusType() string
	GetAuthType() string
	GetAppId() string
	GetAppName() string
	GetLastUpdated() int64
}
