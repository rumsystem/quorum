package consensus

/*

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

var bba_log = logging.Logger("bba")

type AgreementMessage struct {
	Epoch   int
	Message interface{}
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
	acs       *ACS
	groupId   string
	epoch     uint32
	binValues []bool
	sentBvals []bool
	recvBval  map[string]bool
	recvAux   map[string]bool

	done      bool
	output    interface{}
	estimated interface{}
	decision  interface{}

	delayedMessages []*quorumpb.AgreementMsg
	msgCount        int
}

func NewBBA(cfg Config, acs *ACS, groupId string) *BBA {
	if cfg.F == 0 {
		cfg.F = (cfg.N - 1) / 3
	}

	bba := &BBA{
		Config:          cfg,
		acs:             acs,
		groupId:         groupId,
		recvBval:        make(map[string]bool),
		recvAux:         make(map[string]bool),
		sentBvals:       []bool{},
		binValues:       []bool{},
		delayedMessages: []*quorumpb.AgreementMsg{},
	}

	return bba
}

func (b *BBA) InputValue(val bool) error {
	if b.epoch != 0 || b.estimated != nil {
		return nil
	}

	b.estimated = val
	b.sentBvals = append(b.sentBvals, val)

	msg, err := b.makeBValMsg(val)
	if err != nil {
		return err
	}

	SendHbbAgreement(b.groupId, msg)
	return b.handleBvalRequest(msg)
}

func (b *BBA) makeBValMsg(val bool) (*quorumpb.AgreementMsg, error) {
	bval := &quorumpb.BvalReq{
		Value: val,
	}

	bvalb, err := proto.Marshal(bval)
	if err != nil {
		return nil, err
	}

	msg := &quorumpb.AgreementMsg{
		Type:     quorumpb.AgreementMsgType_BVAL_REQ,
		SenderId: b.MyNodeId,
		Epoch:    int64(b.epoch),
		Payload:  bvalb,
	}

	return msg, nil
}

func (b *BBA) makeAuxMsg(val bool) (*quorumpb.AgreementMsg, error) {
	aux := &quorumpb.AuxReq{
		Value: val,
	}

	auxb, err := proto.Marshal(aux)
	if err != nil {
		return nil, err
	}

	msg := &quorumpb.AgreementMsg{
		Type:     quorumpb.AgreementMsgType_AUX_REQ,
		SenderId: b.MyNodeId,
		Epoch:    int64(b.epoch),
		Payload:  auxb,
	}

	return msg, nil
}

func (b *BBA) AcceptInput() bool {
	return b.epoch == 0 && b.estimated == nil
}

//send message to messageCh
func (b *BBA) HandleMessage(msg *quorumpb.AgreementMsg) error {
	b.msgCount++
	if b.done {
		return nil
	}

	if msg.Epoch < int64(b.epoch) {
		return nil
	}

	if msg.Epoch > int64(b.epoch) {
		b.delayedMessages = append(b.delayedMessages, msg)
	}

	switch msg.Type {
	case quorumpb.AgreementMsgType_BVAL_REQ:
		return b.handleBvalRequest(msg)
	case quorumpb.AgreementMsgType_AUX_REQ:
		return b.handleAuxRequest(msg)
	default:
		return fmt.Errorf("Unkonwn BBA message")
	}

}

func (b *BBA) handleBvalRequest(msg *quorumpb.AgreementMsg) error {

	bvalreq := &quorumpb.BvalReq{}
	err := proto.Unmarshal(msg.Payload, bvalreq)
	if err != nil {
		return err
	}
	b.recvBval[msg.SenderId] = bvalreq.Value
	lenBval := b.countBvals(bvalreq.Value)

	//2f + 1node
	if lenBval == 2*b.F+1 {
		wasEmptyBinValues := len(b.binValues) == 0
		b.binValues = append(b.binValues, bvalreq.Value)

		if wasEmptyBinValues {
			//b.addMessage(NewAgreementMsg(int(b.epoch), &AuxRequest{val}))
			auxMsg, err := b.makeAuxMsg(bvalreq.Value)
			if err != nil {
				return err
			}
			SendHbbAgreement(b.groupId, auxMsg)
			b.handleAuxRequest(auxMsg)

		}

		return nil
	}

	if lenBval == b.F+1 && !b.hasSentBval(bvalreq.Value) {
		b.sentBvals = append(b.sentBvals, bvalreq.Value)

		bvalMsg, err := b.makeBValMsg(bvalreq.Value)
		if err != nil {
			return err
		}
		SendHbbAgreement(b.groupId, bvalMsg)
		//b.addMessage(NewAgreementMsg(int(b.epoch), &BValRequest{val}))
		return b.handleBvalRequest(bvalMsg)
	}

	return nil
}

func (b *BBA) handleAuxRequest(msg *quorumpb.AgreementMsg) error {
	auxReq := &quorumpb.AuxReq{}
	err := proto.Unmarshal(msg.Payload, auxReq)
	if err != nil {
		return err
	}
	b.recvAux[msg.SenderId] = auxReq.Value
	return b.tryOutputAgreement()
}

func (b *BBA) tryOutputAgreement() error {
	if len(b.binValues) == 0 {
		return nil
	}

	lenOutputs, values := b.countOutputs()
	if lenOutputs < b.N-b.F {
		return nil
	}

	coin := b.epoch%2 == 0

	if b.done || b.decision != nil && b.decision.(bool) == coin {
		b.done = true
		return nil
	}

	bba_log.Infof("Node (%s) is advancing to next epoch (%d)，received %d aux messages", b.MyNodeId, b.epoch+1, lenOutputs)
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

	//b.addMessage(NewAgreementMsg(int(b.epoch), &BValRequest{estimated}))

	bvalMsg, err := b.makeBValMsg(estimated)
	if err != nil {
		return err
	}

	SendHbbAgreement(b.groupId, bvalMsg)

	for _, que := range b.delayedMessages {
		if err := b.HandleMessage(que); err != nil {
			bba_log.Warn(err)
		}
	}

	b.delayedMessages = []*quorumpb.AgreementMsg{}

	return nil
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

func (b *BBA) Output() interface{} {
	if b.output != nil {
		out := b.output
		b.output = nil
		return out
	}
	return nil
}
*/
