package consensus

import (
	"context"
	"sync"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var ccrmsgsender_log = logging.Logger("ccrsender")

type CCReqSender struct {
	groupId   string
	CurrCCReq *quorumpb.ChangeConsensusReq
	ticker    *time.Ticker
	locker    sync.Mutex
	ctx       context.Context
}

func NewCCReqSender(ctx context.Context, groupId string) *CCReqSender {
	ccrmsgsender_log.Debugf("<%s> NewCCReqSender called", groupId)
	return &CCReqSender{
		groupId:   groupId,
		CurrCCReq: nil,
		ticker:    nil,
		ctx:       ctx,
	}
}
