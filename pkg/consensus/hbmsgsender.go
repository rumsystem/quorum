package consensus

import (
	"sync"
	"time"

	guuid "github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var msg_sender_log = logging.Logger("msender")

const DEFAULT_MSG_SEND_INTEVL = 1 * 1000 //in millseconds

type HBMsgSender struct {
	groupId    string
	interval   int
	pubkey     string
	CurrEpoch  uint64
	CurrHbMsg  *quorumpb.HBMsgv1
	CurrMsgTyp quorumpb.PackageType
	ticker     *time.Ticker
	tickerDone chan bool
	locker     sync.Mutex
}

func NewHBMsgSender(groupId string, epoch uint64, pubkey string, typ quorumpb.PackageType, interval ...int) *HBMsgSender {
	msg_sender_log.Debugf("<%s> NewMsgSender called, Epoch <%d>", groupId, epoch)

	sendingInterval := DEFAULT_MSG_SEND_INTEVL
	if interval != nil {
		sendingInterval = interval[0]
	}

	return &HBMsgSender{
		groupId:    groupId,
		interval:   sendingInterval,
		pubkey:     pubkey,
		CurrEpoch:  epoch,
		CurrHbMsg:  nil,
		CurrMsgTyp: typ,
		ticker:     nil,
		tickerDone: make(chan bool),
	}
}

func (msender *HBMsgSender) SendHBRBCMsg(msg *quorumpb.RBCMsg) error {
	msg_sender_log.Debugf("<%s> SendHBRBCMsg called", msender.groupId)

	rbcb, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	hbmsg := &quorumpb.HBMsgv1{
		MsgId:       guuid.New().String(),
		Epoch:       msender.CurrEpoch,
		PayloadType: quorumpb.HBMsgPayloadType_RBC,
		Payload:     rbcb,
	}

	msender.locker.Lock()
	defer msender.locker.Unlock()

	msender.CurrHbMsg = hbmsg
	msender.startSending()

	return nil
}

func (msender *HBMsgSender) startSending() {
	if msender.ticker != nil {
		msender.tickerDone <- true
	}

	//start new sender ticker
	go func() {
		msg_sender_log.Debugf("<%s> Create ticker <%s>", msender.groupId, msender.CurrHbMsg.MsgId)
		msender.ticker = time.NewTicker(time.Duration(msender.interval) * time.Millisecond)
		for {
			select {
			case <-msender.tickerDone:
				msg_sender_log.Debugf("<%s> old Ticker Done", msender.groupId)
				return
			case <-msender.ticker.C:
				msg_sender_log.Debugf("<%s> tick~ <%s> at <%d>", msender.groupId, msender.CurrHbMsg.MsgId, time.Now().UnixMilli())
				connMgr, err := conn.GetConn().GetConnMgr(msender.groupId)
				if err != nil {
					return
				}
				connMgr.BroadcastHBMsg(msender.CurrHbMsg, msender.CurrMsgTyp)
			}
			msender.ticker.Stop()
		}
	}()
}

func SendHBAABMsg(groupId string, msg *quorumpb.BBAMsg, epoch int64) error {
	//TBD
	return nil
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
