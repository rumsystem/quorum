package consensus

import (
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/NebulousLabs/merkletree"
	"github.com/golang/protobuf/proto"
	"github.com/klauspost/reedsolomon"
	"github.com/rumsystem/quorum/internal/pkg/logging"

	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

const TRXS_TOTAL_SIZE int = 900 * 1024

var rbc_log = logging.Logger("rbc")

type Proofs []*quorumpb.Proof

func (p Proofs) Len() int           { return len(p) }
func (p Proofs) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p Proofs) Less(i, j int) bool { return p[i].Index < p[j].Index }

type RBC struct {
	Config

	groupId    string
	proposerId string //proposerId is pubkey for participated producers node

	acs *ACS //for callback when finished

	numParityShards int
	numDataShards   int

	enc reedsolomon.Encoder

	recvProofs Proofs
	recvReadys map[string]*quorumpb.Ready

	output         []byte
	dataDecodeDone bool
	consusDond     bool
}

// at least 2F + 1 producers are needed
func NewRBC(cfg Config, acs *ACS, groupId, proposerId string) (*RBC, error) {

	// for example F = 1, N = 2 * 1 + 1, 3 producers are needed
	// ecc will encode data bytes into 3 pieces
	// a producer need at least 3 - 1 = 2 pieces to recover data
	var parityShards, dataShards int
	if cfg.N == 1 {
		parityShards = 1
		dataShards = 1
	} else {
		parityShards = cfg.F
		dataShards = cfg.N - cfg.F
	}

	// initial reed solomon codec
	enc, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		return nil, err
	}

	rbc := &RBC{
		Config:          cfg,
		acs:             acs,
		groupId:         groupId,
		proposerId:      proposerId,
		enc:             enc,
		recvProofs:      Proofs{},
		recvReadys:      make(map[string]*quorumpb.Ready),
		numParityShards: parityShards,
		numDataShards:   dataShards,
	}

	return rbc, nil
}

func (r *RBC) HandleMessage(msg *quorumpb.BroadcastMsg) error {
	var err error
	switch msg.Type {
	case quorumpb.BroadcastMsgType_PROOF:
		err = r.handleProofMsg(msg)
	case quorumpb.BroadcastMsgType_READY:
		err = r.handleReadyMsg(msg)
	default:
		err = fmt.Errorf("Invalid RBC protocol %+v", msg)
	}

	return err
}

// when input val in bytes to the rbc instance for myself, the instance will
// 1. seperate bytes to [][]bytes by using reed solomon codec
// 2. make proofReq for each pieces
// 3. broadcast all proofReq via pubsub
func (r *RBC) InputValue(data []byte) error {
	shards, err := makeShards(r.enc, data)
	if err != nil {
		return err
	}

	//create RBC msg for each shards
	reqs, err := r.makeRBCProofMessages(shards)
	if err != nil {
		return err
	}

	// broadcast RBC msg out via pubsub
	for _, req := range reqs {
		err := SendHbbRBC(r.groupId, req)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *RBC) makeRBCProofMessages(shards [][]byte) ([]*quorumpb.BroadcastMsg, error) {
	msgs := make([]*quorumpb.BroadcastMsg, len(shards))

	for i := 0; i < len(msgs); i++ {
		tree := merkletree.New(sha256.New())
		tree.SetIndex(uint64(i))
		for j := 0; j < len(shards); j++ {
			tree.Push(shards[i])
		}
		root, proof, proofIndex, n := tree.Prove()

		//convert group user pubkey to byte
		proposerPubkey := []byte(r.MySignPubkey)
		//sign root(hash) with pubkey
		signature := []byte("FAKE_SIGN")

		payload := &quorumpb.Proof{
			RootHash:       root,
			Proof:          proof,
			Index:          int64(proofIndex),
			Leaves:         int64(n),
			ProposerPubkey: proposerPubkey,
			ProposerSign:   signature,
		}

		payloadb, err := proto.Marshal(payload)
		if err != nil {
			return nil, err
		}

		msgs[i] = &quorumpb.BroadcastMsg{
			SenderPubkey: r.MyNodePubkey,
			Type:         quorumpb.BroadcastMsgType_PROOF,
			Epoch:        int64(r.acs.epoch),
			Payload:      payloadb,
		}
	}

	return msgs, nil
}

func (r *RBC) makeRBCReadyMessage(proof *quorumpb.Proof) (*quorumpb.BroadcastMsg, error) {
	//convert group user pubkey to byte
	proposerPubkey := []byte(r.MySignPubkey)
	//sign root(hash) with pubkey
	signature := []byte("FAKE_SIGN")

	ready := &quorumpb.Ready{
		RootHash:       proof.RootHash,
		ProposerPubkey: proposerPubkey,
		ProposerSign:   signature,
	}

	payloadb, err := proto.Marshal(ready)
	if err != nil {
		return nil, err
	}

	readyMsg := &quorumpb.BroadcastMsg{
		SenderPubkey: r.MyNodePubkey,
		Type:         quorumpb.BroadcastMsgType_READY,
		Payload:      payloadb,
	}

	return readyMsg, nil
}

func (r *RBC) handleProofMsg(msg *quorumpb.BroadcastMsg) error {
	proof := &quorumpb.Proof{}
	err := proto.Unmarshal(msg.Payload, proof)
	if err != nil {
		return err
	}

	//check proposer in producer list
	isInProducerList := false
	for _, nodePubkey := range r.Nodes {
		if nodePubkey == string(proof.ProposerPubkey) {
			isInProducerList = true
			break
		}
	}
	if !isInProducerList {
		return fmt.Errorf("receive proof from non producer %s", proof.ProposerPubkey)
	}

	//check signature
	signOk := true
	if !signOk {
		return fmt.Errorf("invalid proof signature")
	}

	if !validateProof(proof) {
		return fmt.Errorf("Received invalid proof from (%s)", proof.ProposerPubkey)
	}

	//save proof
	r.recvProofs = append(r.recvProofs, proof)

	//if got enough proof, try decode it
	if r.recvProofs.Len() == r.N-r.F {
		err := r.tryDecodeValue()
		if err != nil {
			return err
		}

		//data is ready
		r.dataDecodeDone = true

		//broadcast ready msg
		readyMsg, err := r.makeRBCReadyMessage(proof)
		if err != nil {
			return err
		}

		err = SendHbbRBC(r.groupId, readyMsg)
		if err != nil {
			return err
		}

		//check if we already receive enough readyMsg (N - F -1)
		if len(r.recvReadys) == r.N-r.F-1 {
			r.acs.RbcDone(r.proposerId)
		}
	}

	return nil
}

func (r *RBC) handleReadyMsg(msg *quorumpb.BroadcastMsg) error {
	ready := &quorumpb.Ready{}
	err := proto.Unmarshal(msg.Payload, ready)
	if err != nil {
		return err
	}

	//check if msg sent from producer in list
	isInProducerList := false
	for _, nodePubkey := range r.Nodes {
		if nodePubkey == string(ready.ProposerPubkey) {
			isInProducerList = true
			break
		}
	}
	if !isInProducerList {
		return fmt.Errorf("receive proof from non producer %s", ready.ProposerPubkey)
	}

	//check signature
	signOk := true
	if !signOk {
		return fmt.Errorf("invalid ready signature")
	}

	if _, ok := r.recvReadys[string(ready.ProposerPubkey)]; ok {
		return fmt.Errorf("Received multiple readys from %s", ready.ProposerPubkey)
	}

	r.recvReadys[string(ready.ProposerPubkey)] = ready

	//check if get enough ready
	if len(r.recvReadys) == r.N-r.F && r.dataDecodeDone {
		r.acs.RbcDone(r.proposerId)
	} else {
		//wait till enough
	}

	return nil
}

func validateProof(req *quorumpb.Proof) bool {
	return merkletree.VerifyProof(
		sha256.New(),
		req.RootHash,
		req.Proof,
		uint64(req.Index),
		uint64(req.Leaves))
}

func (r *RBC) tryDecodeValue() error {
	//sort proof by indexId
	sort.Sort(r.recvProofs)

	shards := make([][]byte, r.numParityShards+r.numDataShards)
	for _, p := range r.recvProofs {
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

func (r *RBC) Output() []byte {
	if r.output != nil {
		output := r.output
		r.output = nil
		return output
	}

	return nil
}
