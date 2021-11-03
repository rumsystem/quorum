package pubsubconn

import (
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

type Chain interface {
	HandleTrx(trx *quorumpb.Trx) error
	HandleBlock(block *quorumpb.Block) error
}

type PubSubConn interface {
	JoinChannel(cId string, chain Chain) error
	LeaveChannel(cId string)
	Publish(data []byte) error
}
