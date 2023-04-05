package consensus

import (
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var pcbft_log = logging.Logger("pcbft")

var DEFAULT_CONSENSUS_PROPOSE_PULSE = 1 * 1000 //1s

type PCTask struct {
	Epoch       uint64
	ProposeData []byte
	acsInsts    *PPAcs
}

type PCBft struct {
	Config
	groupId       string
	pp            *MolassesConsensusProposer
	currTask      *PCTask
	currProof     *quorumpb.ConsensusProof
	currProotData []byte
	currEpoch     uint64

	status BftStatus

	tickerLen int64
	tickCnt   int64
	ticker    *time.Ticker

	tickerdone chan bool
	taskdone   chan bool
}

func NewPCBft(cfg Config, pp *MolassesConsensusProposer, tickerLen, tickCnt int64) *PCBft {
	pcbft_log.Debugf("NewPCBft called")
	return &PCBft{
		Config:     cfg,
		groupId:    pp.groupId,
		pp:         pp,
		currTask:   nil,
		currProof:  nil,
		currEpoch:  0,
		status:     IDLE,
		ticker:     nil,
		tickerdone: make(chan bool),
		taskdone:   make(chan bool),
	}
}

func (bft *PCBft) HandleHBMessage(hbmsg *quorumpb.HBMsgv1) error {
	pcbft_log.Debugf("<%s> HandleHBMessage called, Epoch <%d>", bft.groupId, hbmsg.Epoch)
	return nil
}

func (bft *PCBft) AddProof(proof *quorumpb.ConsensusProof) {
	pcbft_log.Debugf("AddProducerProposal called, reqid <%s> ", proof.Req.ReqId)
	bft.currProof = proof
	datab, _ := proto.Marshal(proof)
	bft.currProotData = datab
}

func (bft *PCBft) Start() {
	go func() {
		bft.ticker = time.NewTicker(time.Duration(bft.tickerLen) * time.Millisecond)
		bft.status = RUNNING
		for {
			select {
			case <-bft.tickerdone:
				pcbft_log.Debugf("<%s> TickerDone called", bft.groupId)
				return
			case <-bft.ticker.C:
				pcbft_log.Debugf("<%s> ticker called at <%d>", bft.groupId, time.Now().Nanosecond())
				bft.Propose()
			}
		}
	}()
}

func (bft *PCBft) Stop() {
	if bft.status != RUNNING {
		pcbft_log.Debugf("<%s> bft not running, ignore stop", bft.groupId)
		return
	}

	bft.status = CLOSED
	bft.taskdone <- true
	bft.ticker.Stop()
	bft.tickerdone <- true
	pcbft_log.Debugf("<%s> bft stopped", bft.groupId)
}

func (bft *PCBft) Propose() error {
	bft.currEpoch += 1
	if bft.currEpoch > uint64(bft.tickCnt) {
		//consensus not be done in time
		//stop ticker
		return nil
	}

	acs := NewPPAcs(bft.Config, bft, bft.currEpoch)
	task := &PCTask{
		Epoch:       bft.currEpoch,
		ProposeData: bft.currProotData,
		acsInsts:    acs,
	}

	go func() {
		bft.currTask.acsInsts.InputValue(task.ProposeData)
	}()

	<-bft.taskdone
	return nil
}

func (bft *PCBft) AcsDone(epoch uint64, result map[string][]byte) {
	pcbft_log.Debugf("AcsDone called, epoch <%d>", epoch)
	//give the results back to proposer
	bft.pp.HandleBftDone(epoch, result)
}

func (ppbft *PCBft) HandleHBMsg(hbmsg *quorumpb.HBMsgv1) error {
	pcbft_log.Debugf("HandleHBMsg called, Epoch <%d>", hbmsg.Epoch)
	return nil
}

func (ppbft *PCBft) HandleTimeOut(reqId string) {
}
