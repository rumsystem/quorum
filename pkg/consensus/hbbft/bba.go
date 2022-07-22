package hbbft

import (
	"fmt"
	"sync"

	"github.com/labstack/gommon/log"
)

type AgreementMessage struct {
	Epoch   int
	Message interface{}
}

func NewAgreementMsg(epoch int, msg interface{}) *AgreementMessage {
	return &AgreementMessage{
		Epoch:   epoch,
		Message: msg,
	}
}

type BValRequest struct {
	Value bool
}

type AuxRequest struct {
	Value bool
}

type (
	bbaMessageT struct {
		senderId string
		msg      *AgreementMessage
		err      chan error
	}

	bbaInputT struct {
		value bool
		err   chan error
	}

	delayedMessage struct {
		senderId string
		msg      *AgreementMessage
	}
)

type BBA struct {
	Config
	epoch     uint32
	binValues []bool
	sentBvals []bool
	recvBval  map[string]bool
	recvAux   map[string]bool

	done      bool
	output    interface{}
	estimated interface{}
	decision  interface{}

	delayedMessages []delayedMessage

	lock     sync.RWMutex
	messages []*AgreementMessage

	closeCh   chan struct{}
	inputCh   chan bbaInputT
	messageCh chan bbaMessageT

	msgCount int
}

func NewBBA(cfg Config) *BBA {
	if cfg.F == 0 {
		cfg.F = (cfg.N - 1) / 3
	}

	bba := &BBA{
		Config:          cfg,
		recvBval:        make(map[string]bool),
		recvAux:         make(map[string]bool),
		sentBvals:       []bool{},
		binValues:       []bool{},
		closeCh:         make(chan struct{}),
		inputCh:         make(chan bbaInputT),
		messageCh:       make(chan bbaMessageT),
		messages:        []*AgreementMessage{},
		delayedMessages: []delayedMessage{},
	}

	go bba.run()
	return bba
}

//send input value to inputCh
func (b *BBA) InputValue(val bool) error {
	t := bbaInputT{
		value: val,
		err:   make(chan error),
	}

	b.inputCh <- t
	return <-t.err
}

func (b *BBA) AcceptInput() bool {
	return b.epoch == 0 && b.estimated == nil
}

//send message to messageCh
func (b *BBA) HandleMessage(senderId string, msg *AgreementMessage) error {
	b.msgCount++
	t := bbaMessageT{
		senderId: senderId,
		msg:      msg,
		err:      make(chan error),
	}

	b.messageCh <- t
	return <-t.err
}

func (b *BBA) run() {
	for {
		select {
		case <-b.closeCh:
			return
		case t := <-b.inputCh:
			t.err <- b.inputValue(t.value)
		case t := <-b.messageCh:
			t.err <- b.handleMessage(t.senderId, t.msg)
		}
	}
}

func (b *BBA) addMessage(msg *AgreementMessage) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.messages = append(b.messages, msg)
}

func (b *BBA) stop() {
	close(b.closeCh)
}

func (b *BBA) inputValue(val bool) error {
	if b.epoch != 0 || b.estimated != nil {
		return nil
	}

	b.estimated = val
	b.sentBvals = append(b.sentBvals, val)
	b.addMessage(NewAgreementMsg(int(b.epoch), &BValRequest{val}))
	return b.handleBvalRequest(b.MyNodeId, val)
}

func (b *BBA) handleMessage(senderId string, msg *AgreementMessage) error {
	if b.done {
		return nil
	}

	if msg.Epoch < int(b.epoch) {
		return nil
	}

	if msg.Epoch > int(b.epoch) {
		b.delayedMessages = append(b.delayedMessages, delayedMessage{senderId, msg})
	}

	switch t := msg.Message.(type) {
	case *BValRequest:
		return b.handleBvalRequest(senderId, t.Value)
	case *AuxRequest:
		return b.handleAuxRequest(senderId, t.Value)
	default:
		return fmt.Errorf("Unkonwn BBA message %v", t)
	}
}

func (b *BBA) handleBvalRequest(senderId string, val bool) error {
	b.lock.Lock()
	b.recvBval[senderId] = val
	b.lock.Unlock()
	lenBval := b.countBvals(val)

	//2f + 1node
	if lenBval == 2*b.F+1 {
		wasEmptyBinValues := len(b.binValues) == 0
		b.binValues = append(b.binValues, val)

		if wasEmptyBinValues {
			b.addMessage(NewAgreementMsg(int(b.epoch), &AuxRequest{val}))
			b.handleAuxRequest(b.MyNodeId, val)
		}

		return nil
	}

	if lenBval == b.F+1 && !b.hasSentBval(val) {
		b.sentBvals = append(b.sentBvals, val)
		b.addMessage(NewAgreementMsg(int(b.epoch), &BValRequest{val}))
		return b.handleBvalRequest(b.MyNodeId, val)
	}

	return nil
}

func (b *BBA) handleAuxRequest(senderId string, val bool) error {
	b.lock.Lock()
	b.recvAux[senderId] = val
	b.lock.Unlock()
	b.tryOutputAgreement()
	return nil
}

func (b *BBA) tryOutputAgreement() {
	if len(b.binValues) == 0 {
		return
	}

	lenOutputs, values := b.countOutputs()
	if lenOutputs < b.N-b.F {
		return
	}

	coin := b.epoch%2 == 0

	if b.done || b.decision != nil && b.decision.(bool) == coin {
		b.done = true
		return
	}

	//fmt.Printf("Node (%s) is advancing to next epoch (%d)，received %d aux messages", b.MyNodeId, b.epoch + 1, lenlenOutputs)
	b.advanceEpoch()

	if len(values) != 1 {
		b.estimated = coin
	} else {
		b.estimated = values[0]

		if b.decision == nil && values[0] == coin {
			b.output = values[0]
			b.decision = values[0]
			b.msgCount = 0
		}
	}

	estimated := b.estimated.(bool)
	b.sentBvals = append(b.sentBvals, estimated)
	b.addMessage(NewAgreementMsg(int(b.epoch), &BValRequest{estimated}))

	for _, que := range b.delayedMessages {
		if err := b.handleMessage(que.senderId, que.msg); err != nil {
			log.Warn(err)
		}
	}

	b.delayedMessages = []delayedMessage{}
}

// clear all and advance epoch
func (b *BBA) advanceEpoch() {
	b.binValues = []bool{}
	b.sentBvals = []bool{}
	b.recvAux = make(map[string]bool)
	b.recvBval = make(map[string]bool)
	b.epoch++
}

func (b *BBA) countOutputs() (int, []bool) {
	m := map[bool]string{}
	for senderId, val := range b.recvAux {
		m[val] = senderId
	}

	vals := []bool{}
	for _, val := range b.binValues {
		if _, ok := m[val]; ok {
			vals = append(vals, val)
		}
	}
	return len(b.recvAux), vals
}

func (b *BBA) countBvals(ok bool) int {
	b.lock.RLock()
	defer b.lock.RUnlock()

	n := 0
	for _, val := range b.recvBval {
		if val == ok {
			n++
		}
	}
	return n
}

func (b *BBA) hasSentBval(val bool) bool {
	for _, ok := range b.sentBvals {
		if ok == val {
			return true
		}
	}
	return false
}

func (b *BBA) Messages() []*AgreementMessage {
	b.lock.RLock()
	msgs := b.messages
	b.lock.RUnlock()

	b.lock.Lock()
	defer b.lock.Unlock()
	b.messages = []*AgreementMessage{}
	return msgs
}

func (b *BBA) Output() interface{} {
	if b.output != nil {
		out := b.output
		b.output = nil
		return out
	}
	return nil
}
