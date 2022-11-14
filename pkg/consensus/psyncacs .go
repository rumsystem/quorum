package consensus

import (
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var psync_acs_log = logging.Logger("pacs")

type PSyncACS struct {
	Config
	bft          *PSyncBft
	SessionId    string
	rbcInstances map[string]*PSyncRBC
	rbcOutput    map[string]bool
	rbcResults   map[string][]byte
}

func NewPSyncACS(cfg Config, bft *PSyncBft, sid string) *PSyncACS {
	psync_acs_log.Debugf("<%s> NewPSyncACS called, SessionId <%s>", bft.PSyncer.groupId, sid)

	acs := &PSyncACS{
		Config:       cfg,
		bft:          bft,
		SessionId:    sid,
		rbcInstances: make(map[string]*PSyncRBC),
		rbcOutput:    make(map[string]bool),
		rbcResults:   make(map[string][]byte),
	}

	for _, id := range cfg.Nodes {
		acs.rbcInstances[id], _ = NewPSyncRBC(cfg, acs, bft.PSyncer.groupId, id)
	}

	return acs
}

// give input value to
func (a *PSyncACS) InputValue(val []byte) error {
	psync_acs_log.Debugf("<%s> InputValue called", a.bft.PSyncer.groupId)

	rbc, ok := a.rbcInstances[a.MySignPubkey]
	if !ok {
		return fmt.Errorf("could not find rbc instance (%s)", a.MySignPubkey)
	}

	return rbc.InputValue(val)
}

// rbc for proposerIs finished
func (a *PSyncACS) RbcDone(proposerPubkey string) {
	psync_acs_log.Debugf("<%s> RbcDone called, SessionId <%s>", a.bft.PSyncer.groupId, a.SessionId)

	a.rbcOutput[proposerPubkey] = true

	//check if all rbc instance output
	if len(a.rbcOutput) == a.N-a.F {
		trx_acs_log.Debugf("all RBC done, call acs")
		// all rbc done, get all rbc results, send them back to BFT
		for _, rbcInst := range a.rbcInstances {
			//load all rbc results
			a.rbcResults[rbcInst.proposerPubkey] = rbcInst.Output()
		}

		//call hbb to get result
		a.bft.AcsDone(a.SessionId, a.rbcResults)
	} else {
		psync_acs_log.Debugf("Wait for all RBC done")
		return
	}
}

func (a *PSyncACS) HandleMessage(hbmsg *quorumpb.HBMsgv1) error {
	psync_acs_log.Debugf("<%s> HandleMessage called,  SessionId <%s>", a.bft.PSyncer.groupId, hbmsg.SessionId)

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
