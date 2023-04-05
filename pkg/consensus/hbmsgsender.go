package consensus

import (
	"sync"
	"time"

	guuid "github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

//var hbmsgsender_log = logging.Logger("hbmsender")

const DEFAULT_HB_MSG_SEND_INTEVL = 1 * 1000 //in millseconds

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
	//hbmsgsender_log.Debugf("<%s> NewHBMsgSender called, Epoch <%d>", groupId, epoch)

	sendingInterval := DEFAULT_HB_MSG_SEND_INTEVL
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
	//hbmsgsender_log.Debugf("<%s> SendHBRBCMsg called", msender.groupId)

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

func (msender *HBMsgSender) StopSending() {

	msender.locker.Lock()
	defer msender.locker.Unlock()
	if msender.ticker != nil {
		msender.tickerDone <- true
	}
}

func (msender *HBMsgSender) startSending() {
	if msender.ticker != nil {
		msender.tickerDone <- true
	}

	//start new sender ticker
	go func() {
		//hbmsgsender_log.Debugf("<%s> Create ticker <%s>", msender.groupId, msender.CurrHbMsg.MsgId)
		msender.ticker = time.NewTicker(time.Duration(msender.interval) * time.Millisecond)
		for {
			select {
			case <-msender.tickerDone:
				//hbmsgsender_log.Debugf("<%s> old Ticker Done", msender.groupId)
				return
			case <-msender.ticker.C:
				//hbmsgsender_log.Debugf("<%s> tick~ <%s> at <%d>", msender.groupId, msender.CurrHbMsg.MsgId, time.Now().UnixMilli())
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

// TBD
func (msender *HBMsgSender) SendHBAABMsg(groupId string, msg *quorumpb.BBAMsg, epoch int64) error {
	return nil
}
