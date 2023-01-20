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
	Epoch        int64
	rbcInstances map[string]*TrxRBC
	rbcOutput    map[string]bool
	rbcResults   map[string][]byte
}

func NewTrxACS(cfg Config, bft *TrxBft, epoch int64) *TrxACS {
	trx_acs_log.Infof("NewTrxACS called epoch <%d>", epoch)

	acs := &TrxACS{
		Config:       cfg,
		bft:          bft,
		Epoch:        epoch,
		rbcInstances: make(map[string]*TrxRBC),
		rbcOutput:    make(map[string]bool),
		rbcResults:   make(map[string][]byte),
	}

	for _, id := range cfg.Nodes {
		acs.rbcInstances[id], _ = NewTrxRBC(cfg, acs, bft.producer.groupId, id)
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
		//this only works for 3 nodes!!
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
	trx_acs_log.Infof("HandleMessage called, Epoch <%d>", hbmsg.Epoch)
	//unmarshall BLOCK payload
	blockMsg := &quorumpb.HBBlockMsg{}
	err := proto.Unmarshal(hbmsg.Payload, blockMsg)
	if err != nil {
		return err
	}

	switch blockMsg.MsgType {
	case quorumpb.HBBlockMsgType_RBC:
		return a.handleRbc(blockMsg.Payload)
	case quorumpb.HBBlockMsgType_BBA:
		return a.handleBba(blockMsg.Payload)
	default:
		return fmt.Errorf("received unknown type BlockMsg <%s>", blockMsg.MsgType.String())
	}
}

func (a *TrxACS) handleRbc(payload []byte) error {
	trx_acs_log.Infof("handleRbc called, Epoch <%d>", a.Epoch)

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
			return fmt.Errorf("could not find rbc instance to handle proof from <%s>, original propose <%d>", echo.EchoProviderPubkey, echo.OriginalProposerPubkey)
		}
		return rbc.handleEchoMsg(echo)
	case quorumpb.RBCMsgType_READY:
		ready := &quorumpb.Ready{}
		err := proto.Unmarshal(rbcMsg.Payload, ready)
		if err != nil {
			return err
		}
		rbc, ok := a.rbcInstances[ready.ReadyProviderPubkey]
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
