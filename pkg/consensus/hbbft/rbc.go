package hbbft

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/NebulousLabs/merkletree"
	"github.com/golang/protobuf/proto"
	"github.com/klauspost/reedsolomon"
	"github.com/rumsystem/quorum/internal/pkg/logging"

	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

var rbc_log = logging.Logger("rbc")

type proofs []*quorumpb.ProofReq

func (p proofs) Len() int           { return len(p) }
func (p proofs) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p proofs) Less(i, j int) bool { return p[i].Index < p[j].Index }

type RBC struct {
	Config

	groupId    string
	proposerId string

	numParityShards int
	numDataShards   int

	enc reedsolomon.Encoder

	recvReadys map[string]*quorumpb.ReadyReq
	recvEchos  map[string]*quorumpb.EchoReq

	echoSent      bool
	readySent     bool
	outputDecoded bool

	output []byte
}

//proposerId is uuid for other participated nodes
func NewRBC(cfg Config, groupId, proposerId string) (*RBC, error) {

	// calculate failer node
	if cfg.F == 0 {
		cfg.F = (cfg.N - 1) / 3
	}
	// calculate how to make data shards (with enc codec)
	var (
		parityShards = 2 * cfg.F
		dataShards   = cfg.N - parityShards
	)

	// initial reed solomon codec
	enc, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		return nil, err
	}

	rbc := &RBC{
		Config:          cfg,
		groupId:         groupId,
		proposerId:      proposerId,
		enc:             enc,
		recvEchos:       make(map[string]*quorumpb.EchoReq),
		recvReadys:      make(map[string]*quorumpb.ReadyReq),
		numParityShards: parityShards,
		numDataShards:   dataShards,
	}

	return rbc, nil
}

func (r *RBC) HandleMessage(msg *quorumpb.BroadcastMsg) error {
	var err error
	switch msg.Type {
	case quorumpb.BroadcastMsgType_PROOF_REQ:
		err = r.handleProofRequest(msg)
	case quorumpb.BroadcastMsgType_ECHO_REQ:
		err = r.handleEchoRequest(msg)
	case quorumpb.BroadcastMsgType_READY_REQ:
		err = r.handleEchoRequest(msg)
	default:
		err = fmt.Errorf("Invalid RBC protocol %+v", msg)
	}

	return err
}

func (r *RBC) InputValue(data []byte) ([]*quorumpb.BroadcastMsg, error) {
	shards, err := makeShards(r.enc, data)
	if err != nil {
		return nil, err
	}

	//create RBC msg for each shards
	reqs, err := r.makeRBCProofMessage(shards)
	if err != nil {
		return nil, err
	}

	// first rbc msg is mine
	if err := r.handleProofRequest(reqs[0]); err != nil {
		return nil, err
	}

	return reqs[1:], nil
}

func (r *RBC) makeRBCProofMessage(shards [][]byte) ([]*quorumpb.BroadcastMsg, error) {
	msgs := make([]*quorumpb.BroadcastMsg, len(shards))

	for i := 0; i < len(msgs); i++ {
		tree := merkletree.New(sha256.New())
		tree.SetIndex(uint64(i))
		for j := 0; j < len(shards); j++ {
			tree.Push(shards[i])
		}
		root, proof, proofIndex, n := tree.Prove()
		payload := &quorumpb.ProofReq{
			RootHash: root,
			Proof:    proof,
			Index:    int64(proofIndex),
			Leaves:   int64(n),
		}

		payloadb, err := proto.Marshal(payload)
		if err != nil {
			return nil, err
		}

		msgs[i] = &quorumpb.BroadcastMsg{
			Type:     quorumpb.BroadcastMsgType_PROOF_REQ,
			SenderId: r.MyNodeId,
			Payload:  payloadb,
		}
	}

	return msgs, nil
}

func (r *RBC) makeRBCEchoMessage(proof *quorumpb.ProofReq) (*quorumpb.BroadcastMsg, error) {

	echoReq := &quorumpb.EchoReq{
		Req: proof,
	}

	echoReqb, err := proto.Marshal(echoReq)
	if err != nil {
		return nil, err
	}

	echoMsg := &quorumpb.BroadcastMsg{
		Type:     quorumpb.BroadcastMsgType_ECHO_REQ,
		SenderId: r.MyNodeId,
		Payload:  echoReqb,
	}

	return echoMsg, nil
}

func (r *RBC) makeRBCReadyMessage(echo *quorumpb.EchoReq) (*quorumpb.BroadcastMsg, error) {
	readyReq := &quorumpb.ReadyReq{
		RootHash: echo.Req.RootHash,
	}

	readyB, err := proto.Marshal(readyReq)
	if err != nil {
		return nil, err
	}

	readyMsg := &quorumpb.BroadcastMsg{
		Type:     quorumpb.BroadcastMsgType_READY_REQ,
		SenderId: r.MyNodeId,
		Payload:  readyB,
	}

	return readyMsg, nil
}

func (r *RBC) handleProofRequest(msg *quorumpb.BroadcastMsg) error {
	proofReq := &quorumpb.ProofReq{}
	err := proto.Unmarshal(msg.Payload, proofReq)
	if err != nil {
		return err
	}

	if msg.SenderId != r.proposerId {
		return fmt.Errorf("Receiving proof from (%s) that is not from the proposing node (%s)", msg.SenderId, r.proposerId)
	}

	if r.echoSent {
		return fmt.Errorf("Received proof from (%s) more than once", msg.SenderId)
	}

	if !validateProof(proofReq) {
		return fmt.Errorf("Received invalid proof from (%s)", msg.SenderId)
	}

	r.echoSent = true

	echoMsg, err := r.makeRBCEchoMessage(proofReq)
	if err != nil {
		return err
	}

	//add message to msg queue
	// r.messages = append(r.messages, echoMsg)
	SendHbbRBC(r.groupId, echoMsg)

	return r.handleEchoRequest(echoMsg)
}

func (r *RBC) handleEchoRequest(msg *quorumpb.BroadcastMsg) error {
	echoReq := &quorumpb.EchoReq{}
	err := proto.Unmarshal(msg.Payload, echoReq)
	if err != nil {
		return err
	}

	if _, ok := r.recvEchos[msg.SenderId]; ok {
		return fmt.Errorf("Received multiple echos from (%s)", msg.SenderId)
	}

	if !validateProof(echoReq.Req) {
		return fmt.Errorf("Received invalid proof from (%s)", msg.SenderId)
	}

	r.recvEchos[msg.SenderId] = echoReq
	if r.readySent || r.countEcho(echoReq.Req.RootHash) < r.N-r.F {
		return r.tryDecodeValue(echoReq.Req.RootHash)
	}

	r.readySent = true

	readyMsg, err := r.makeRBCReadyMessage(echoReq)
	//r.messages = append(r.messages, readyMsg)
	SendHbbRBC(r.groupId, readyMsg)

	return r.handleReadyRequest(readyMsg)
}

func (r *RBC) handleReadyRequest(msg *quorumpb.BroadcastMsg) error {
	if _, ok := r.recvReadys[msg.SenderId]; ok {
		return fmt.Errorf("Received multiple readys from %s", msg.SenderId)
	}

	readyReq := &quorumpb.ReadyReq{}
	err := proto.Unmarshal(msg.Payload, readyReq)
	if err != nil {
		return err
	}

	r.recvReadys[msg.SenderId] = readyReq

	if r.countReady(readyReq.RootHash) == r.F+1 && !r.readySent {
		r.readySent = true
		//r.messages = append(r.messages, msg)
		SendHbbRBC(r.groupId, msg)
	}

	return r.tryDecodeValue(readyReq.RootHash)
}

func validateProof(req *quorumpb.ProofReq) bool {
	return merkletree.VerifyProof(
		sha256.New(),
		req.RootHash,
		req.Proof,
		uint64(req.Index),
		uint64(req.Leaves))
}

func (r *RBC) tryDecodeValue(hash []byte) error {
	if r.outputDecoded || r.countReady(hash) <= 2*r.F || r.countEcho(hash) <= r.F {
		return nil
	}

	r.outputDecoded = true
	var prfs proofs

	for _, echo := range r.recvEchos {
		prfs = append(prfs, echo.Req)
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
		if bytes.Compare(hash, e.Req.RootHash) == 0 {
			n++
		}
	}
	return n
}

func (r *RBC) countReady(hash []byte) int {
	n := 0
	for _, e := range r.recvReadys {
		if bytes.Compare(hash, e.RootHash) == 0 {
			n++
		}
	}
	return n
}

func (r *RBC) Output() []byte {
	if r.output != nil {
		output := r.output
		r.output = nil
		return output
	}

	return nil
}
