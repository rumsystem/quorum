package consensus

import (
	"sync"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var pprmsgsender_log = logging.Logger("pprsender")

const DEFAULT_PP_REQ_SEND_INTEVL = 1 * 1000 //in millseconds

type PPReqSender struct {
	groupId    string
	interval   int
	pubkey     string
	CurrPPReq  *quorumpb.ProducerProposalReq
	ticker     *time.Ticker
	tickerDone chan bool
	locker     sync.Mutex
}

func NewPPReqSender(groupId string, pubkey string, interval ...int) *PPReqSender {
	pprmsgsender_log.Debugf("<%s> NewPPReqSender called", groupId)

	sendingInterval := DEFAULT_PP_REQ_SEND_INTEVL
	if interval != nil {
		sendingInterval = interval[0]
	}

	return &PPReqSender{
		groupId:    groupId,
		interval:   sendingInterval,
		pubkey:     pubkey,
		CurrPPReq:  nil,
		ticker:     nil,
		tickerDone: make(chan bool),
	}
}

func (msender *PPReqSender) SendPPReq(msg *quorumpb.ProducerProposalReq) error {
	pprmsgsender_log.Debugf("<%s> SendPPReq called", msender.groupId)

	msender.locker.Lock()
	defer msender.locker.Unlock()

	msender.CurrPPReq = msg
	msender.startSending()

	return nil
}

func (msender *PPReqSender) startSending() {
	if msender.ticker != nil {
		msender.tickerDone <- true
	}

	//start new sender ticker
	go func() {
		pprmsgsender_log.Debugf("<%s> Create ticker <%s>", msender.groupId, msender.CurrPPReq.ReqId)
		msender.ticker = time.NewTicker(time.Duration(msender.interval) * time.Millisecond)
		for {
			select {
			case <-msender.tickerDone:
				pprmsgsender_log.Debugf("<%s> old Ticker Done", msender.groupId)
				return
			case <-msender.ticker.C:
				pprmsgsender_log.Debugf("<%s> tick~ <%s> at <%d>", msender.groupId, msender.CurrPPReq.ReqId, time.Now().UnixMilli())
				connMgr, err := conn.GetConn().GetConnMgr(msender.groupId)
				if err != nil {
					return
				}
				connMgr.BroadcastPPReq(msender.CurrPPReq)
			}
			msender.ticker.Stop()
		}
	}()
}

/*
func SendPSyncReqMsg(groupId string, msg *quorumpb.PSyncReq) error {
	connMgr, err := conn.GetConn().GetConnMgr(groupId)
	if err != nil {
		return err
	}

	msgB, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	psyncReq := &quorumpb.PSyncMsg{
		MsgType: quorumpb.PSyncMsgType_PSYNC_REQ,
		Payload: msgB,
	}

	return connMgr.BroadcastPSyncMsg(psyncReq)
}

func SendPSyncRespMsg(groupId string, msg *quorumpb.PSyncResp) error {
	connMgr, err := conn.GetConn().GetConnMgr(groupId)
	if err != nil {
		return err
	}

	msgB, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	psyncResp := &quorumpb.PSyncMsg{
		MsgType: quorumpb.PSyncMsgType_PSYNC_RESP,
		Payload: msgB,
	}

	return connMgr.BroadcastPSyncMsg(psyncResp)
}

*/
