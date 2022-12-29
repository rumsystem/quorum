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
	epoch        int64
	rbcInstances map[string]*TrxRBC
	rbcOutput    map[string]bool
	rbcResults   map[string][]byte
}

func NewTrxACS(cfg Config, bft *TrxBft, epoch int64) *TrxACS {
	trx_acs_log.Infof("NewTrxACS called epoch <%d>", epoch)

	acs := &TrxACS{
		Config:       cfg,
		bft:          bft,
		epoch:        epoch,
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

	rbc, ok := a.rbcInstances[a.MySignPubkey]
	if !ok {
		return fmt.Errorf("could not find rbc instance (%s)", a.MySignPubkey)
	}

	return rbc.InputValue(val)
}

// rbc for proposerIs finished
func (a *TrxACS) RbcDone(proposerPubkey string) {
	trx_acs_log.Infof("RbcDone called, Epoch <%d>", a.epoch)

	a.rbcOutput[proposerPubkey] = true

	//check if all rbc instance output
	if len(a.rbcOutput) == a.N-a.f {
		trx_acs_log.Debugf("enough RBC done, call acs")
		// all rbc done, get all rbc results, send them back to BFT
		for rbcInst, _ := range a.rbcOutput {
			//load all valid rbc results
			a.rbcResults[rbcInst] = a.rbcInstances[rbcInst].Output()
		}

		//call hbb to get result
		a.bft.AcsDone(a.epoch, a.rbcResults)
	} else {
		trx_acs_log.Debugf("Wait for enough RBC done")
		return
	}
}

func (a *TrxACS) HandleMessage(hbmsg *quorumpb.HBMsgv1) error {
	trx_acs_log.Infof("HandleMessage called, Epoch <%d>", hbmsg.Epoch)
	switch hbmsg.MsgType {
	case quorumpb.HBBMsgType_BROADCAST:
		broadcastMsg := &quorumpb.BroadcastMsg{}
		err := proto.Unmarshal(hbmsg.Payload, broadcastMsg)
		if err != nil {
			return err
		}
		switch broadcastMsg.Type {
		case quorumpb.BroadcastMsgType_PROOF:
			proof := &quorumpb.Proof{}
			err := proto.Unmarshal(broadcastMsg.Payload, proof)
			if err != nil {
				return err
			}
			rbc, ok := a.rbcInstances[proof.ProposerPubkey]
			if !ok {
				return fmt.Errorf("could not find rbc instance to handle proof for (%s)", proof.ProposerPubkey)
			}
			return rbc.handleProofMsg(proof)
		case quorumpb.BroadcastMsgType_READY:
			ready := &quorumpb.Ready{}
			err := proto.Unmarshal(broadcastMsg.Payload, ready)
			if err != nil {
				return err
			}
			rbc, ok := a.rbcInstances[ready.ProofProviderPubkey]
			if !ok {
				return fmt.Errorf("could not find rbc instance to handle ready for (%s)", ready.ProofProviderPubkey)
			}
			return rbc.handleReadyMsg(ready)

		default:
			return fmt.Errorf("received unknown broadcast message (%v)", broadcastMsg.Type)
		}
	default:
		return fmt.Errorf("received unknown hbmsg <%s> type (%v)", hbmsg.MsgId, hbmsg.MsgType)
	}
}
