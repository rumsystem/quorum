package consensus

import (
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var ptacs_log = logging.Logger("ptacs")

type PTAcs struct {
	Config
	bft        *PTBft
	Epoch      uint64
	rbcInsts   map[string]*PTRbc
	rbcOutput  map[string]bool
	rbcResults map[string][]byte
}

func NewPTACS(cfg Config, bft *PTBft, epoch uint64) *PTAcs {
	ptacs_log.Infof("NewTrxACS called epoch <%d>", epoch)

	acs := &PTAcs{
		Config:     cfg,
		bft:        bft,
		Epoch:      epoch,
		rbcInsts:   make(map[string]*PTRbc),
		rbcOutput:  make(map[string]bool),
		rbcResults: make(map[string][]byte),
	}

	for _, rbcInstPubkey := range cfg.Nodes {
		acs.rbcInsts[rbcInstPubkey], _ = NewPTRBC(cfg, acs, bft.producer.groupId, cfg.MyPubkey, rbcInstPubkey)
	}

	return acs
}

func (a *PTAcs) InputValue(val []byte) error {
	ptacs_log.Info("InputValue called")

	rbc, ok := a.rbcInsts[a.MyPubkey]
	if !ok {
		return fmt.Errorf("could not find rbc instance <%s>", a.MyPubkey)
	}

	return rbc.InputValue(val)
}

// rbc for proposerIs finished
func (a *PTAcs) RbcDone(proposerPubkey string) {
	ptacs_log.Infof("RbcDone called, RBC <%s> finished", proposerPubkey)
	a.rbcOutput[proposerPubkey] = true
	if len(a.rbcOutput) == a.N-a.f {
		//ptacs_log.Debugf("enough RBC done, consensus needed<%d>", a.N-a.f)
		//this only works when producer nodes equals to 3!!
		//TBD:should add BBA here
		//1. set all NOT finished RBC to false
		//2. start BBA process till finished
		for rbcInst, _ := range a.rbcOutput {
			//load all valid rbc results
			a.rbcResults[rbcInst] = a.rbcInsts[rbcInst].Output()
		}

		//call hbb to get result
		a.bft.AcsDone(a.Epoch, a.rbcResults)
	} else {
		//ptacs_log.Debugf("Wait for enough RBC done")
		return
	}
}

func (a *PTAcs) HandleHBMessage(hbmsg *quorumpb.HBMsgv1) error {
	//	ptacs_log.Debugf("<%d> HandleMessage called, Epoch <%d>", hbmsg.Epoch, a.Epoch)

	switch hbmsg.PayloadType {
	case quorumpb.HBMsgPayloadType_RBC:
		return a.handleRbcMsg(hbmsg.Payload)
	case quorumpb.HBMsgPayloadType_BBA:
		return a.handleBbaMsg(hbmsg.Payload)
	default:
		return fmt.Errorf("received unknown type msg <%s>", hbmsg.PayloadType.String())
	}
}

func (a *PTAcs) handleRbcMsg(payload []byte) error {
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
		//	ptacs_log.Debugf("epoch <%d> : INIT_PROPOSE: sender <%s> receiver <%s>", a.Epoch, initp.ProposerPubkey, initp.RecvNodePubkey)
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

func (a *PTAcs) handleBbaMsg(payload []byte) error {
	//TBD
	//Implement BBA
	return nil
}
