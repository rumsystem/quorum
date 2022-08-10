package consensus

import (
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var acs_log = logging.Logger("acs")

type ACS struct {
	Config
	groupId      string
	bft          *Bft
	epoch        uint64
	rbcInstances map[string]*RBC
	rbcOutput    map[string]bool
	rbcResults   map[string][]byte
}

func NewACS(cfg Config, bft *Bft, epoch uint64) *ACS {
	acs := &ACS{
		Config:       cfg,
		groupId:      bft.groupId,
		bft:          bft,
		epoch:        epoch,
		rbcInstances: make(map[string]*RBC),
		rbcOutput:    make(map[string]bool),
		rbcResults:   make(map[string][]byte),
	}

	for _, id := range cfg.Nodes {
		acs.rbcInstances[id], _ = NewRBC(cfg, acs, bft.groupId, id)
	}

	return acs
}

//give input value to
func (a *ACS) InputValue(val []byte) error {
	rbc, ok := a.rbcInstances[a.MyNodePubkey]
	if !ok {
		return fmt.Errorf("could not find rbc instance (%s)", a.MyNodePubkey)
	}

	return rbc.InputValue(val)
}

//rbc for proposerIs finished
func (a *ACS) RbcDone(proposerId string) {
	a.rbcOutput[proposerId] = true

	//check if all rbc instance output
	if len(a.rbcOutput) == a.N {
		// all rbc done, get all rbc results, send them back to BFT
		for _, rbcInst := range a.rbcInstances {
			//load all rbc results
			a.rbcResults[rbcInst.proposerId] = rbcInst.Output()
		}

		//call hbb to get result
		a.bft.AcsDone(a.epoch, a.rbcResults)
	} else {
		//continue waiting
		return
	}
}
func (a *ACS) HandleMessage(msg *quorumpb.HBMsg) error {
	switch msg.MsgType {
	case quorumpb.HBBMsgType_BROADCAST:
		return a.processBroadcast(msg)
	default:
		return fmt.Errorf("received unknown message (%v)", msg.MsgType)
	}
}

func (a *ACS) processBroadcast(msg *quorumpb.HBMsg) error {
	broadcastMsg := &quorumpb.BroadcastMsg{}
	err := proto.Unmarshal(msg.Payload, broadcastMsg)
	if err != nil {
		return err
	}

	rbc, ok := a.rbcInstances[broadcastMsg.SenderPubkey]
	if !ok {
		return fmt.Errorf("could not find rbc instance for (%s)", broadcastMsg.SenderPubkey)
	}

	return rbc.HandleMessage(broadcastMsg)
}
