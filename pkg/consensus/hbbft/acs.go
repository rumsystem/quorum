package hbbft

import (
	"fmt"
)

type ACSMessage struct {
	ProposerID string
	Payload    interface{}
}

type ACS struct {
	Config
	rbcInstances map[string]*RBC
	bbaInstances map[string]*BBA
	rbcResults   map[string][]byte
	bbaResults   map[string]bool
	output       map[string][]byte
	messageQue   *messageQue
	decided      bool

	closeCh   chan struct{}
	inputCh   chan acsInputTuple
	messageCh chan acsMessageTuple
}

type (
	acsMessageTuple struct {
		senderID string
		msg      *ACSMessage
		err      chan error
	}

	acsInputResponse struct {
		rbcMessages []*BroadcastMessage
		acsMessages []*ACSMessage
		err         error
	}

	acsInputTuple struct {
		value    []byte
		response chan acsInputResponse
	}
)

func NewACS(cfg Config) *ACS {
	if cfg.F == 0 {
		cfg.F = (cfg.N - 1) / 3
	}
	acs := &ACS{
		Config:       cfg,
		rbcInstances: make(map[string]*RBC),
		bbaInstances: make(map[string]*BBA),
		rbcResults:   make(map[string][]byte),
		bbaResults:   make(map[string]bool),
		messageQue:   newMessageQue(),
		closeCh:      make(chan struct{}),
		inputCh:      make(chan acsInputTuple),
		messageCh:    make(chan acsMessageTuple),
	}

	for _, id := range cfg.Nodes {
		acs.rbcInstances[id], _ = NewRBC(cfg, id)
		acs.bbaInstances[id] = NewBBA(cfg)
	}
	go acs.run()
	return acs
}

func (a *ACS) InputValue(val []byte) error {
	t := acsInputTuple{
		value:    val,
		response: make(chan acsInputResponse),
	}
	a.inputCh <- t
	resp := <-t.response
	return resp.err
}

func (a *ACS) HandleMessage(senderID string, msg *ACSMessage) error {
	t := acsMessageTuple{
		senderID: senderID,
		msg:      msg,
		err:      make(chan error),
	}
	a.messageCh <- t
	return <-t.err
}

func (a *ACS) handleMessage(senderID string, msg *ACSMessage) error {
	switch t := msg.Payload.(type) {
	case *AgreementMessage:
		return a.handleAgreement(senderID, msg.ProposerID, t)
	case *BroadcastMessage:
		return a.handleBroadcast(senderID, msg.ProposerID, t)
	default:
		return fmt.Errorf("received unknown message (%v)", t)
	}
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

func (a *ACS) inputValue(data []byte) error {
	rbc, ok := a.rbcInstances[a.MyNodeId]
	if !ok {
		return fmt.Errorf("could not find rbc instance (%d)", a.MyNodeId)
	}
	reqs, err := rbc.InputValue(data)
	if err != nil {
		return err
	}
	if len(reqs) != a.N-1 {
		return fmt.Errorf("expecting (%d) proof messages got (%d)", a.N, len(reqs))
	}
	for i, id := range stringsWithout(a.Nodes, a.MyNodeId) {
		a.messageQue.addMessage(&ACSMessage{a.MyNodeId, reqs[i]}, id)
	}
	for _, msg := range rbc.Messages() {
		a.addMessage(a.MyNodeId, msg)
	}
	if output := rbc.Output(); output != nil {
		a.rbcResults[a.MyNodeId] = output
		a.processAgreement(a.MyNodeId, func(bba *BBA) error {
			if bba.AcceptInput() {
				return bba.InputValue(true)
			}
			return nil
		})
	}
	return nil
}

func (a *ACS) stop() {
	close(a.closeCh)
}

func (a *ACS) run() {
	for {
		select {
		case <-a.closeCh:
			return
		case t := <-a.inputCh:
			err := a.inputValue(t.value)
			t.response <- acsInputResponse{err: err}
		case t := <-a.messageCh:
			t.err <- a.handleMessage(t.senderID, t.msg)
		}
	}
}

func (a *ACS) handleAgreement(senderId, proposerId string, msg *AgreementMessage) error {
	return a.processAgreement(proposerId, func(bba *BBA) error {
		return bba.HandleMessage(senderId, msg)
	})
}

func (a *ACS) handleBroadcast(senderId, proposerId string, msg *BroadcastMessage) error {
	return a.processBroadcast(proposerId, func(rbc *RBC) error {
		return rbc.HandleMessage(senderId, msg)
	})
}

func (a *ACS) processBroadcast(proposerId string, fun func(rbc *RBC) error) error {
	rbc, ok := a.rbcInstances[proposerId]
	if !ok {
		return fmt.Errorf("could not find rbc instance for (%d)", proposerId)
	}
	if err := fun(rbc); err != nil {
		return err
	}
	for _, msg := range rbc.Messages() {
		a.addMessage(proposerId, msg)
	}
	if output := rbc.Output(); output != nil {
		a.rbcResults[proposerId] = output
		return a.processAgreement(proposerId, func(bba *BBA) error {
			if bba.AcceptInput() {
				return bba.InputValue(true)
			}
			return nil
		})
	}
	return nil
}

func (a *ACS) processAgreement(proposerId string, fun func(bba *BBA) error) error {
	bba, ok := a.bbaInstances[proposerId]
	if !ok {
		return fmt.Errorf("could not find bba instance for (%d)", proposerId)
	}
	if bba.done {
		return nil
	}
	if err := fun(bba); err != nil {
		return err
	}
	for _, msg := range bba.Messages() {
		a.addMessage(proposerId, msg)
	}
	// Check if we got an output.
	if output := bba.Output(); output != nil {
		if _, ok := a.bbaResults[proposerId]; ok {
			return fmt.Errorf("multiple bba results for (%d)", proposerId)
		}
		a.bbaResults[proposerId] = output.(bool)
		// When received 1 from at least (N - f) instances of BA, provide input 0.
		// to each other instance of BBA that has not provided his input yet.
		if output.(bool) && a.countTruthyAgreements() == a.N-a.F {
			for id, bba := range a.bbaInstances {
				if bba.AcceptInput() {
					if err := bba.InputValue(false); err != nil {
						return err
					}
					for _, msg := range bba.Messages() {
						a.addMessage(id, msg)
					}
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
