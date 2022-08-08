package hbbft

import (
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var acs_log = logging.Logger("acs")

type ACSMessage struct {
	ProposerID string
	Payload    interface{}
}

type ACS struct {
	Config
	groupId      string
	rbcInstances map[string]*RBC
	bbaInstances map[string]*BBA
	rbcResults   map[string][]byte
	bbaResults   map[string]bool
	output       map[string][]byte
	decided      bool
}

func NewACS(cfg Config, groupId string) *ACS {
	if cfg.F == 0 {
		cfg.F = (cfg.N - 1) / 3
	}
	acs := &ACS{
		Config:       cfg,
		groupId:      groupId,
		rbcInstances: make(map[string]*RBC),
		bbaInstances: make(map[string]*BBA),
		rbcResults:   make(map[string][]byte),
		bbaResults:   make(map[string]bool),
	}

	for _, id := range cfg.Nodes {
		acs.rbcInstances[id], _ = NewRBC(cfg, acs, groupId, id)
		acs.bbaInstances[id] = NewBBA(cfg, acs, groupId)
	}

	return acs
}

//give input value to
func (a *ACS) InputValue(val []byte) error {
	rbc, ok := a.rbcInstances[a.MyNodeId]
	if !ok {
		return fmt.Errorf("could not find rbc instance (%d)", a.MyNodeId)
	}

	return rbc.InputValue(val)
}

//rbc for proposerIs finished
func (a *ACS) RbcDone(proposerId string) {
	//rbcInst := a.rbcInstances[proposerId]

	/*
		if output := rbcInst.Output(); output != nil {
			a.rbcResults[a.MyNodeId] = output
			a.processAgreement(a.MyNodeId, func(bba *BBA) error {
				if bba.AcceptInput() {
					return bba.InputValue(true)
				}
				return nil
			})
		}
	*/
}

func (a *ACS) AcsDone(proposerId string) {

}

func (a *ACS) HandleMessage(msg *quorumpb.HBMsg) error {
	switch msg.MsgType {
	case quorumpb.HBBMsgType_AGREEMENT:
		return a.processAgreement(msg)
	case quorumpb.HBBMsgType_BROADCAST:
		return a.processBroadcast(msg)
	default:
		return fmt.Errorf("received unknown message (%v)", msg.MsgType)
	}
}

func (a *ACS) processBroadcast(msg *quorumpb.HBMsg) error {

	rbcMsg := &quorumpb.BroadcastMsg{}
	err := proto.Unmarshal(msg.Payload, rbcMsg)
	if err != nil {
		return err
	}

	rbc, ok := a.rbcInstances[rbcMsg.SenderId]
	if !ok {
		return fmt.Errorf("could not find rbc instance for (%d)", rbcMsg.SenderId)
	}

	return rbc.HandleMessage(rbcMsg)
}

func (a *ACS) processAgreement(msg *quorumpb.HBMsg) error {

	bbaMsg := &quorumpb.AgreementMsg{}
	err := proto.Unmarshal(msg.Payload, bbaMsg)
	if err != nil {
		return err
	}

	bba, ok := a.bbaInstances[bbaMsg.SenderId]
	if !ok {
		return fmt.Errorf("could not find bba instance for (%d)", bbaMsg.SenderId)
	}

	if bba.done {
		return nil
	}

	err = bba.HandleMessage(bbaMsg)
	if err != nil {
		return err
	}

	// Check if we got an output.
	if output := bba.Output(); output != nil {
		if _, ok := a.bbaResults[bbaMsg.SenderId]; ok {
			return fmt.Errorf("multiple bba results for (%d)", bbaMsg.SenderId)
		}
		a.bbaResults[bbaMsg.SenderId] = output.(bool)
		// When received 1 from at least (N - f) instances of BA, provide input 0.
		// to each other instance of BBA that has not provided his input yet.
		if output.(bool) && a.countTruthyAgreements() == a.N-a.F {
			for id, bba := range a.bbaInstances {
				if bba.AcceptInput() {
					if err := bba.InputValue(false); err != nil {
						return err
					}

					/*
						for _, msg := range bba.Messages() {
							a.addMessage(id, msg)
						}
					*/
					if output := bba.Output(); output != nil {
						a.bbaResults[id] = output.(bool)
					}
				}
			}
		}
		a.tryCompleteAgreement()
	}
	return nil
}

func (a *ACS) Output() map[string][]byte {
	if a.output != nil {
		out := a.output
		a.output = nil
		return out
	}
	return nil
}

func (a *ACS) Done() bool {
	agreementsDone := true
	for _, bba := range a.bbaInstances {
		if !bba.done {
			agreementsDone = false
		}
	}
	return agreementsDone && a.messageQue.len() == 0
}

func (a *ACS) tryCompleteAgreement() {
	if a.decided || a.countTruthyAgreements() < a.N-a.F {
		return
	}
	if len(a.bbaResults) < a.N {
		return
	}
	// At this point all bba instances have provided their output.
	nodesThatProvidedTrue := []string{}
	for id, ok := range a.bbaResults {
		if ok {
			nodesThatProvidedTrue = append(nodesThatProvidedTrue, id)
		}
	}
	bcResults := make(map[string][]byte)
	for _, id := range nodesThatProvidedTrue {
		val, _ := a.rbcResults[id]
		bcResults[id] = val
	}
	if len(nodesThatProvidedTrue) == len(bcResults) {
		a.output = bcResults
		a.decided = true
	}
}

func (a *ACS) addMessage(from string, msg interface{}) {
	for _, id := range stringsWithout(a.Nodes, a.MyNodeId) {
		a.messageQue.addMessage(&ACSMessage{from, msg}, id)
	}
}

// countTruthyAgreements returns the number of truthy received agreement messages.
func (a *ACS) countTruthyAgreements() int {
	n := 0
	for _, ok := range a.bbaResults {
		if ok {
			n++
		}
	}
	return n
}

func stringsWithout(s []string, val string) []string {
	dest := []string{}
	for i := 0; i < len(s); i++ {
		if s[i] != val {
			dest = append(dest, s[i])
		}
	}
	return dest
}
