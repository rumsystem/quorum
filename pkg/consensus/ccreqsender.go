package consensus

import (
	"sync"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var ccrmsgsender_log = logging.Logger("ccrsender")

const DEFAULT_CC_REQ_SEND_INTEVL = 1 * 1000 //in millseconds

type CCReqSender struct {
	groupId    string
	interval   int
	pubkey     string
	CurrCCReq  *quorumpb.ChangeConsensusReq
	ticker     *time.Ticker
	tickerDone chan bool
	locker     sync.Mutex
}

func NewCCReqSender(groupId string, pubkey string, interval ...int) *CCReqSender {
	ccrmsgsender_log.Debugf("<%s> NewCCReqSender called", groupId)

	sendingInterval := DEFAULT_CC_REQ_SEND_INTEVL
	if interval != nil {
		sendingInterval = interval[0]
	}

	return &CCReqSender{
		groupId:    groupId,
		interval:   sendingInterval,
		pubkey:     pubkey,
		CurrCCReq:  nil,
		ticker:     nil,
		tickerDone: make(chan bool),
	}
}

func (msender *CCReqSender) SendCCReq(req *quorumpb.ChangeConsensusReq) error {
	ccrmsgsender_log.Debugf("<%s> SendCCReq called", msender.groupId)

	msender.locker.Lock()
	defer msender.locker.Unlock()

	msender.CurrCCReq = req
	msender.startSending()

	return nil
}

func (msender *CCReqSender) startSending() {
	if msender.ticker != nil {
		msender.tickerDone <- true
	}

	//start new sender ticker
	go func() {
		ccrmsgsender_log.Debugf("<%s> create ticker <%s>", msender.groupId, msender.CurrCCReq.ReqId)
		msender.ticker = time.NewTicker(time.Duration(msender.interval) * time.Millisecond)
		for {
			select {
			case <-msender.tickerDone:
				ccrmsgsender_log.Debugf("<%s> old Ticker Done", msender.groupId)
				return
			case <-msender.ticker.C:
				ccrmsgsender_log.Debugf("<%s> tick~ <%s> at <%d>", msender.groupId, msender.CurrCCReq.ReqId, time.Now().UnixMilli())
				connMgr, err := conn.GetConn().GetConnMgr(msender.groupId)
				if err != nil {
					return
				}
				connMgr.BroadcastPPReq(msender.CurrCCReq)
			}
			msender.ticker.Stop()
		}
	}()
}
