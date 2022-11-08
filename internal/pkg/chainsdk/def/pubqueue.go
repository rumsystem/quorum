package def

import quorumpb "github.com/rumsystem/quorum/pkg/pb"

type PublishQueueIface interface {
	TrxEnqueue(groupId string, trx *quorumpb.Trx) error
}
