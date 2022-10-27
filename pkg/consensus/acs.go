package consensus

import (
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var acs_log = logging.Logger("acs")

type ACS struct {
	Config
	groupId      string
	bft          *Bft
	epoch        int64
	rbcInstances map[string]*RBC
	rbcOutput    map[string]bool
	rbcResults   map[string][]byte
}

func NewACS(cfg Config, bft *Bft, epoch int64) *ACS {
	acs_log.Infof("NewACS called epoch <%d>", epoch)

	acs := &ACS{
		Config:       cfg,
		bft:          bft,
		epoch:        epoch,
		rbcInstances: make(map[string]*RBC),
		rbcOutput:    make(map[string]bool),
		rbcResults:   make(map[string][]byte),
	}

	for _, id := range cfg.Nodes {
		acs.rbcInstances[id], _ = NewRBC(cfg, acs, bft.producer.groupId, id)
	}

	return acs
}

// give input value to
func (a *ACS) InputValue(val []byte) error {
	rbc, ok := a.rbcInstances[a.MySignPubkey]
	if !ok {
		return fmt.Errorf("could not find rbc instance (%s)", a.MySignPubkey)
	}

	return rbc.InputValue(val)
}

// rbc for proposerIs finished
func (a *ACS) RbcDone(proposerPubkey string) {
	acs_log.Infof("RbcDone called, Epoch <%d>", a.epoch)
	a.rbcOutput[proposerPubkey] = true

	//check if all rbc instance output
	if len(a.rbcOutput) == a.N-a.F {
		acs_log.Debugf("all RBC done, call acs")
		// all rbc done, get all rbc results, send them back to BFT
		for _, rbcInst := range a.rbcInstances {
			//load all rbc results
			a.rbcResults[rbcInst.proposerPubkey] = rbcInst.Output()
		}

		//call hbb to get result
		a.bft.AcsDone(a.epoch, a.rbcResults)
	} else {
		acs_log.Debugf("Wait for all RBC done")
		return
	}
}

func (a *ACS) HandleMessage(hbmsg *quorumpb.HBMsgv1) error {
	acs_log.Infof("HandleMessage called, Epoch <%d>", hbmsg.Epoch)
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
