package consensus

import (
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var pcacs_log = logging.Logger("pcacs")

type PPAcs struct {
	Config
	Epoch      uint64
	rbcInsts   map[string]*PCRbc
	rbcOutput  map[string]bool
	rbcResults map[string][]byte

	chAcsDone chan *AcsResult
}

func NewPPAcs(groupId, nodename string, cfg Config, epoch uint64, chAcsDone chan *AcsResult) *PPAcs {
	pcacs_log.Debugf("NewPPAcs called, epoch <%d>", epoch)

	acs := &PPAcs{
		Config:     cfg,
		Epoch:      epoch,
		rbcInsts:   make(map[string]*PCRbc),
		rbcOutput:  make(map[string]bool),
		rbcResults: make(map[string][]byte),

		chAcsDone: chAcsDone,
	}

	for _, nodeID := range cfg.Nodes {
		acs.rbcInsts[nodeID], _ = NewPCRbc(cfg, acs, groupId, nodename, cfg.MyPubkey, nodeID)
	}

	return acs
}

// give input value to
func (a *PPAcs) InputValue(val []byte) error {
	pcacs_log.Debug("InputValue called")

	rbc, ok := a.rbcInsts[a.MyPubkey]
	if !ok {
		return fmt.Errorf("could not find rbc instance <%s>", a.MyPubkey)
	}

	return rbc.InputValue(val)
}

// rbc for proposerIs finished
func (a *PPAcs) RbcDone(proposerPubkey string) {
	pcacs_log.Debugf("RbcDone called, RBC <%s> finished", proposerPubkey)
	a.rbcOutput[proposerPubkey] = true

	if len(a.rbcOutput) == a.N-a.f {
		ptacs_log.Debugf("enough RBC finished", a.N-a.f)
		for rbcInst, _ := range a.rbcOutput {
			a.rbcResults[rbcInst] = a.rbcInsts[rbcInst].Output()
		}

		//notify acs done
		a.chAcsDone <- &AcsResult{
			epoch:  a.Epoch,
			result: a.rbcResults,
		}
	} else {
		pcacs_log.Debugf("Wait for enough RBC done")
	}
}

func (a *PPAcs) HandleHBMessage(hbmsg *quorumpb.HBMsgv1) error {
	ptacs_log.Debugf("<%d> HandleMessage called, Epoch <%d>", hbmsg.Epoch, a.Epoch)

	switch hbmsg.PayloadType {
	case quorumpb.HBMsgPayloadType_RBC:
		return a.handleRbcMsg(hbmsg.Payload)
	case quorumpb.HBMsgPayloadType_BBA:
		return a.handleBbaMsg(hbmsg.Payload)
	default:
		return fmt.Errorf("received unknown type msg <%s>", hbmsg.PayloadType.String())
	}
}

func (a *PPAcs) handleRbcMsg(payload []byte) error {
	//ptacs_log.Debugf("handleRbc called, Epoch <%d>", a.Epoch)

	//cast payload to RBC message
	rbcMsg := &quorumpb.RBCMsg{}
	err := proto.Unmarshal(payload, rbcMsg)
	if err != nil {
		return err
	}

	switch rbcMsg.Type {
	case quorumpb.RBCMsgType_INIT_PROPOSE:
		initp := &quorumpb.InitPropose{}
		err := proto.Unmarshal(rbcMsg.Payload, initp)
		if err != nil {
			return err
		}
		ptacs_log.Debugf("epoch <%d> : INIT_PROPOSE: sender <%s> receiver <%s>", a.Epoch, initp.ProposerPubkey, initp.RecvNodePubkey)
		if initp.RecvNodePubkey != a.MyPubkey {
			ptacs_log.Debugf("not for me")
			return nil
		}

		rbc, ok := a.rbcInsts[initp.ProposerPubkey]
		if !ok {
			return fmt.Errorf("could not find rbc instance to handle InitPropose form <%s>", initp.ProposerPubkey)
		}

		return rbc.handleInitProposeMsg(initp)
	case quorumpb.RBCMsgType_ECHO:
		echo := &quorumpb.Echo{}
		err := proto.Unmarshal(rbcMsg.Payload, echo)
		if err != nil {
			return err
		}
		//give the ECHO msg to original proposer
		rbc, ok := a.rbcInsts[echo.OriginalProposerPubkey]
		if !ok {
			return fmt.Errorf("could not find rbc instance to handle proof from <%s>, original propose <%s>", echo.EchoProviderPubkey, echo.OriginalProposerPubkey)
		}
		return rbc.handleEchoMsg(echo)
	case quorumpb.RBCMsgType_READY:
		ready := &quorumpb.Ready{}
		err := proto.Unmarshal(rbcMsg.Payload, ready)
		if err != nil {
			return err
		}
		rbc, ok := a.rbcInsts[ready.OriginalProposerPubkey]
		if !ok {
			return fmt.Errorf("could not find rbc instance to handle ready from <%s>", ready.ReadyProviderPubkey)
		}
		return rbc.handleReadyMsg(ready)

	default:
		return fmt.Errorf("received unknown rbc message, type (%s)", rbcMsg.Type)
	}
}

// TBD
func (a *PPAcs) handleBbaMsg(payload []byte) error {
	return nil
}
