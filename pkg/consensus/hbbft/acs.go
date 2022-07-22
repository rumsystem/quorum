package hbbft

import (
	"fmt"
)

// ACSMessage represents a message sent between nodes in the ACS protocol.
type ACSMessage struct {
	// Unique identifier of the "proposing" node.
	ProposerID string
	// Actual payload beeing sent.
	Payload interface{}
}

// ACS implements the Asynchronous Common Subset protocol.
// ACS assumes a network of N nodes that send signed messages to each other.
// There can be f faulty nodes where (3 * f < N).
// Each participating node proposes an element for inlcusion. The protocol
// guarantees that all of the good nodes output the same set, consisting of
// at least (N -f) of the proposed values.
//
// Algorithm:
// ACS creates a Broadcast algorithm for each of the participating nodes.
// At least (N -f) of these will eventually output the element proposed by that
// node. ACS will also create and BBA instance for each participating node, to
// decide whether that node's proposed element should be inlcuded in common set.
// Whenever an element is received via broadcast, we imput "true" into the
// corresponding BBA instance. When (N-f) BBA instances have decided true we
// input false into the remaining ones, where we haven't provided input yet.
// Once all BBA instances have decided, ACS returns the set of all proposed
// values for which the decision was truthy.
type ACS struct {
	// Config holds the ACS configuration.
	Config
	// Mapping of node ids and their rbc instance.
	rbcInstances map[string]*RBC
	// Mapping of node ids and their bba instance.
	bbaInstances map[string]*BBA
	// Results of the Reliable Broadcast.
	rbcResults map[string][]byte
	// Results of the Binary Byzantine Agreement.
	bbaResults map[string]bool
	// Final output of the ACS.
	output map[string][]byte
	// Que of ACSMessages that need to be broadcasted after each received
	// and processed a message.
	messageQue *messageQue
	// Whether this ACS instance has already has decided output or not.
	decided bool

	// control flow tuples for internal channel communication.
	closeCh   chan struct{}
	inputCh   chan acsInputTuple
	messageCh chan acsMessageTuple
}

// Control flow structure for internal channel communication. Allowing us to
// avoid the use of mutexes and eliminates race conditions.
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

// NewACS returns a new ACS instance configured with the given Config and node
// ids.
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
	// Create all the instances for the participating nodes
	for _, id := range cfg.Nodes {
		acs.rbcInstances[id], _ = NewRBC(cfg, id)
		acs.bbaInstances[id] = NewBBA(cfg)
	}
	go acs.run()
	return acs
}

// InputValue sets the input value for broadcast and returns an initial set of
// Broadcast and ACS Messages to be broadcasted in the network.
func (a *ACS) InputValue(val []byte) error {
	t := acsInputTuple{
		value:    val,
		response: make(chan acsInputResponse),
	}
	a.inputCh <- t
	resp := <-t.response
	return resp.err
}

// HandleMessage handles incoming messages to ACS and redirects them to the
// appropriate sub(protocol) instance.
func (a *ACS) HandleMessage(senderID string, msg *ACSMessage) error {
	t := acsMessageTuple{
		senderID: senderID,
		msg:      msg,
		err:      make(chan error),
	}
	a.messageCh <- t
	return <-t.err
}

// handleMessage handles incoming messages to ACS and redirects them to the
// appropriate sub(protocol) instance.
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

// Output will return the output of the ACS instance. If the output was not nil
// then it will return the output else nil. Note that after consuming the output
// its will be set to nil forever.
func (a *ACS) Output() map[string][]byte {
	if a.output != nil {
		out := a.output
		a.output = nil
		return out
	}
	return nil
}

// Done returns true whether ACS has completed its agreements and cleared its
// messageQue.
func (a *ACS) Done() bool {
	agreementsDone := true
	for _, bba := range a.bbaInstances {
		if !bba.done {
			agreementsDone = false
		}
	}
	return agreementsDone && a.messageQue.len() == 0
}

// inputValue sets the input value for broadcast and returns an initial set of
// Broadcast and ACS Messages to be broadcasted in the network.
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

// handleAgreement processes the received AgreementMessage from sender (sid)
// for a value proposed by the proposing node (pid).
func (a *ACS) handleAgreement(senderId, proposerId string, msg *AgreementMessage) error {
	return a.processAgreement(proposerId, func(bba *BBA) error {
		return bba.HandleMessage(senderId, msg)
	})
}

// handleBroadcast processes the received BroadcastMessage.
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
