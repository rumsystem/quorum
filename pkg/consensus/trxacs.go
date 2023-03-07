package consensus

import (
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var trx_acs_log = logging.Logger("tacs")

type TrxACS struct {
	Config
	bft          *TrxBft
	Epoch        uint64
	rbcInstances map[string]*TrxRBC
	rbcOutput    map[string]bool
	rbcResults   map[string][]byte
}

func NewTrxACS(cfg Config, bft *TrxBft, epoch uint64) *TrxACS {
	trx_acs_log.Infof("NewTrxACS called epoch <%d>", epoch)

	acs := &TrxACS{
		Config:       cfg,
		bft:          bft,
		Epoch:        epoch,
		rbcInstances: make(map[string]*TrxRBC),
		rbcOutput:    make(map[string]bool),
		rbcResults:   make(map[string][]byte),
	}

	for _, rbcInstPubkey := range cfg.Nodes {
		acs.rbcInstances[rbcInstPubkey], _ = NewTrxRBC(cfg, acs, bft.producer.groupId, cfg.MyPubkey, rbcInstPubkey)
	}

	return acs
}

// give input value to
func (a *TrxACS) InputValue(val []byte) error {
	trx_acs_log.Info("InputValue called")

	rbc, ok := a.rbcInstances[a.MyPubkey]
	if !ok {
		return fmt.Errorf("could not find rbc instance (%s)", a.MyPubkey)
	}

	return rbc.InputValue(val)
}

// rbc for proposerIs finished
func (a *TrxACS) RbcDone(proposerPubkey string) {
	trx_acs_log.Infof("RbcDone called, Epoch <%d>", a.Epoch)
	a.rbcOutput[proposerPubkey] = true
	if len(a.rbcOutput) == a.N-a.f {
		trx_acs_log.Debugf("enough RBC done for consensus <%d>", a.N-a.f)
		//this only works when producer nodes equals to 3!!
		//TBD:should add BBA here
		//1. set all NOT finished RBC to false
		//2. start BBA process till finished
		for rbcInst, _ := range a.rbcOutput {
			//load all valid rbc results
			a.rbcResults[rbcInst] = a.rbcInstances[rbcInst].Output()
		}

		//call hbb to get result
		a.bft.AcsDone(a.Epoch, a.rbcResults)
	} else {
		trx_acs_log.Debugf("Wait for enough RBC done")
		return
	}
}

func (a *TrxACS) HandleMessage(hbmsg *quorumpb.HBMsgv1) error {
	trx_acs_log.Debugf("<%d> HandleMessage called, Epoch <%d>", hbmsg.Epoch, a.Epoch)

	switch hbmsg.PayloadType {
	case quorumpb.HBMsgPayloadType_RBC:
		return a.handleRbc(hbmsg.Payload)
	case quorumpb.HBMsgPayloadType_BBA:
		return a.handleBba(hbmsg.Payload)
	default:
		return fmt.Errorf("received unknown type BlockMsg <%s>", hbmsg.PayloadType.String())
	}
}

func (a *TrxACS) handleRbc(payload []byte) error {
	//trx_acs_log.Debugf("handleRbc called, Epoch <%d>", a.Epoch)

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
		trx_acs_log.Debugf("epoch <%d> : INIT_PROPOSE: sender <%s> receiver <%s>", a.Epoch, initp.ProposerPubkey, initp.RecvNodePubkey)
		if initp.RecvNodePubkey != a.MyPubkey {
			trx_acs_log.Debugf("not for me")
			return nil
		}

		rbc, ok := a.rbcInstances[initp.ProposerPubkey]
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
		rbc, ok := a.rbcInstances[echo.OriginalProposerPubkey]
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
		rbc, ok := a.rbcInstances[ready.OriginalProposerPubkey]
		if !ok {
			return fmt.Errorf("could not find rbc instance to handle ready from <%s>", ready.ReadyProviderPubkey)
		}
		return rbc.handleReadyMsg(ready)

	default:
		return fmt.Errorf("received unknown rbc message, type (%s)", rbcMsg.Type)
	}
}

func (a *TrxACS) handleBba(payload []byte) error {
	//TBD
	//Implement BBA
	return nil
}
