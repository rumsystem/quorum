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
	groupId      string
	epochLenInMs uint64
	epochCnt     uint64
	CurrCCReq    *quorumpb.ChangeConsensusReq
	ticker       *time.Ticker
	tickerDone   chan bool
	locker       sync.Mutex

	proposer   *MolassesConsensusProposer
	sendingCnt uint64
}

func NewCCReqSender(groupId string, epochLenInMs, epochCnt uint64, proposer *MolassesConsensusProposer) *CCReqSender {
	ccrmsgsender_log.Debugf("<%s> NewCCReqSender called", groupId)

	return &CCReqSender{
		groupId:      groupId,
		epochLenInMs: epochLenInMs,
		epochCnt:     epochCnt,
		proposer:     proposer,
		CurrCCReq:    nil,
		ticker:       nil,
		tickerDone:   make(chan bool),
		sendingCnt:   0,
	}
}

func (msender *CCReqSender) SendCCReq(req *quorumpb.ChangeConsensusReq) error {
	ccrmsgsender_log.Debugf("<%s> SendCCReq called", msender.groupId)

	msender.locker.Lock()
	defer msender.locker.Unlock()

	msender.CurrCCReq = req
	msender.sendingCnt = 0
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
		msender.ticker = time.NewTicker(time.Duration(msender.epochLenInMs) * time.Millisecond)
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
				msender.sendingCnt += 1

				if msender.sendingCnt >= msender.epochCnt {
					ccrmsgsender_log.Debugf("<%s> CCReqSender stop sending <%s> at <%d>", msender.groupId, msender.CurrCCReq.ReqId, time.Now().UnixMilli())
					msender.proposer.bft.HandleTimeOut(msender.CurrCCReq.ReqId)
					msender.tickerDone <- true
				}
			}
			msender.ticker.Stop()
		}
	}()
}
