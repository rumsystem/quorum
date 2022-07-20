package hbbft

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/NebulousLabs/merkletree"
	"github.com/klauspost/reedsolomon"
)

type BroadcastMessage struct {
	Payload interface{}
}

type ProofRequest struct {
	RootHash []byte
	// Proof[0] will containt the actual data.
	Proof         [][]byte
	Index, Leaves int
}

type EchoRequest struct {
	ProofRequest
}

type ReadyRequest struct {
	RootHash []byte
}

type proofs []ProofRequest

func (p proofs) Len() int           { return len(p) }
func (p proofs) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p proofs) Less(i, j int) bool { return p[i].Index < p[j].Index }

type (
	rbcMessageT struct {
		senderId string
		msg      *BroadcastMessage
		err      chan error
	}

	rbcInputResp struct {
		message []*BroadcastMessage
		err     error
	}

	rbcInputT struct {
		value    []byte
		response chan rbcInputResp
	}
)

type RBC struct {
	config     Config
	proposerId string

	enc reedsolomon.Encoder

	recvReadys map[string][]byte
	recvEchos  map[string]*EchoRequest

	numParityShards int
	numDataShards   int

	messages []*BroadcastMessage

	echoSent      bool
	readySent     bool
	outputDecoded bool

	output    []byte
	closeCh   chan struct{}
	inputCh   chan rbcInputT
	messageCh chan rbcMessageT
}

func Init(config Config, proposerId string, consusNum int) (*RBC, error) {
	canFailedNode := (consusNum - 1) / 3
	parityShareds := 2 * canFailedNode
	dataShards := consusNum - parityShareds

	enc, err := reedsolomon.New(dataShards, parityShareds)
	if err != nil {
		return nil, err
	}

	rbc := &RBC{
		config:          config,
		proposerId:      proposerId,
		enc:             enc,
		recvEchos:       make(map[string]*EchoRequest),
		recvReadys:      make(map[string][]byte),
		numParityShards: parityShareds,
		numDataShards:   dataShards,
		messages:        []*BroadcastMessage{},
		closeCh:         make(chan struct{}),
		inputCh:         make(chan rbcInputT),
		messageCh:       make(chan rbcMessageT),
	}

	go rbc.run()
	return rbc, nil
}

func (r *RBC) InputValue(data []byte) ([]*BroadcastMessage, error) {
	t := rbcInputT{
		value:    data,
		response: make(chan rbcInputResp),
	}

	r.inputCh <- t
	resp := <-t.response
	return resp.message, resp.err
}

func (r *RBC) HandleMessage(senderId string, msg *BroadcastMessage) error {
	t := rbcMessageT{
		senderId: senderId,
		msg:      msg,
		err:      make(chan error),
	}
	r.messageCh <- t
	return <-t.err
}

func (r *RBC) stop() {
	close(r.closeCh)
}

func (r *RBC) run() {
	for {
		select {
		case <-r.closeCh:
			return
		case t := <-r.inputCh:
			msgs, err := r.inputValue(t.value)
			t.response <- rbcInputResp{
				message: msgs,
				err:     err,
			}
		case t := <-r.messageCh:
			t.err <- r.handleMessage(t.senderId, t.msg)
		}
	}
}

func (r *RBC) inputValue(data []byte) ([]*BroadcastMessage, error) {
	shards, err := makeShards(r.enc, data)
	if err != nil {
		return nil, err
	}

	reqs, err := makeBroadcastMessage(shards)
	if err != nil {
		return nil, err
	}

	proof := reqs[0].Payload.(*ProofRequest)
	if err := r.handleProofRequest(r.config.MyNodeId, proof); err != nil {
		return nil, err
	}

	return reqs[1:], nil
}

func (r *RBC) handleMessage(senderId string, msg *BroadcastMessage) error {
	switch t := msg.Payload.(type) {
	case *ProofRequest:
		return r.handleProofRequest(senderId, t)
	case *EchoRequest:
		return r.handleEchoRequest(senderId, t)
	case *ReadyRequest:
		return r.handleReadyRequest(senderId, t)
	default:
		return fmt.Errorf("Invalid RBC protocol %+v", msg)
	}
}

func (r *RBC) handleProofRequest(senderId string, req *ProofRequest) error {
	if senderId != r.proposerId {
		return fmt.Errorf("Receiving proof from (%s) that is not from the proposing node (%s)", senderId, r.proposerId)
	}

	if r.echoSent {
		return fmt.Errorf("Received proof from (%s) more than once", senderId)
	}

	if !validateProof(req) {
		return fmt.Errorf("Received invalid proof from (%s)", senderId)
	}

	r.echoSent = true
	echo := &EchoRequest{*req}
	r.messages = append(r.messages, &BroadcastMessage{echo})
	return r.handleEchoRequest(r.config.MyNodeId, echo)
}

func (r *RBC) handleEchoRequest(senderId string, req *EchoRequest) error {
	if _, ok := r.recvEchos[senderId]; ok {
		return fmt.Errorf("Received multiple echos from (%s)", senderId)
	}

	if !validateProof(&req.ProofRequest) {
		return fmt.Errorf("Received invalid proof from (%s)", senderId)
	}

	r.recvEchos[senderId] = req
	if r.readySent || r.countEcho(req.RootHash) < r.config.N-r.config.F {
		return r.tryDecodeValue(req.RootHash)
	}

	r.readySent = true
	ready := &ReadyRequest{req.RootHash}
	r.messages = append(r.messages, &BroadcastMessage{ready})
	return r.handleReadyRequest(r.config.MyNodeId, ready)
}

func (r *RBC) handleReadyRequest(senderId string, req *ReadyRequest) error {
	if _, ok := r.recvReadys[senderId]; ok {
		return fmt.Errorf("Received multiple readys from %s", senderId)
	}
	r.recvReadys[senderId] = req.RootHash

	if r.countReady(req.RootHash) == r.config.F+1 && !r.readySent {
		r.readySent = true
		ready := &ReadyRequest{req.RootHash}
		r.messages = append(r.messages, &BroadcastMessage{ready})
	}

	return r.tryDecodeValue(req.RootHash)
}

func (r *RBC) tryDecodeValue(hash []byte) error {
	if r.outputDecoded || r.countReady(hash) <= 2*r.config.F || r.countEcho(hash) <= r.config.F {
		return nil
	}

	r.outputDecoded = true
	var prfs proofs

	for _, echo := range r.recvEchos {
		prfs = append(prfs, echo.ProofRequest)
	}

	sort.Sort(prfs)

	shards := make([][]byte, r.numParityShards+r.numDataShards)
	for _, p := range prfs {
		shards[p.Index] = p.Proof[0]
	}

	if err := r.enc.Reconstruct(shards); err != nil {
		return nil
	}

	var value []byte
	for _, data := range shards[:r.numDataShards] {
		value = append(value, data...)
	}

	r.output = value
	return nil
}

func makeProofRequests(shards [][]byte) ([]*ProofRequest, error) {
	reqs := make([]*ProofRequest, len(shards))
	for i := 0; i < len(reqs); i++ {
		tree := merkletree.New(sha256.New())
		tree.SetIndex(uint64(i))
		for j := 0; j < len(shards); j++ {
			tree.Push(shards[i])
		}

		root, proof, proofIndex, n := tree.Prove()
		reqs[i] = &ProofRequest{
			RootHash: root,
			Proof:    proof,
			Index:    int(proofIndex),
			Leaves:   int(n),
		}
	}

	return reqs, nil
}

func makeBroadcastMessage(shards [][]byte) ([]*BroadcastMessage, error) {
	msgs := make([]*BroadcastMessage, len(shards))

	for i := 0; i < len(msgs); i++ {
		tree := merkletree.New(sha256.New())
		tree.SetIndex(uint64(i))
		for j := 0; j < len(shards); j++ {
			tree.Push(shards[i])
		}
		root, proof, proofIndex, n := tree.Prove()
		msgs[i] = &BroadcastMessage{
			Payload: &ProofRequest{
				RootHash: root,
				Proof:    proof,
				Index:    int(proofIndex),
				Leaves:   int(n),
			},
		}
	}
	return msgs, nil
}

func validateProof(req *ProofRequest) bool {
	return merkletree.VerifyProof(
		sha256.New(),
		req.RootHash,
		req.Proof,
		uint64(req.Index),
		uint64(req.Leaves))
}

func makeShards(enc reedsolomon.Encoder, data []byte) ([][]byte, error) {
	shards, err := enc.Split(data)
	if err != nil {
		return nil, err
	}

	if err := enc.Encode(shards); err != nil {
		return nil, err
	}

	return shards, nil
}

func (r *RBC) countEcho(hash []byte) int {
	n := 0
	for _, e := range r.recvEchos {
		if bytes.Compare(hash, e.RootHash) == 0 {
			n++
		}
	}
	return n
}

func (r *RBC) countReady(hash []byte) int {
	n := 0
	for _, e := range r.recvReadys {
		if bytes.Compare(hash, e) == 0 {
			n++
		}
	}
	return n
}
