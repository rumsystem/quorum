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

	groupId        string
	proposerPubkey string //proposerPubkey is pubkey for participated witnesses node

	acs *ACS //for callback when finished

	numParityShards int
	numDataShards   int

	enc reedsolomon.Encoder

	recvProofs Proofs
	recvReadys map[string]*quorumpb.Ready

	output         []byte
	dataDecodeDone bool
	consenusDone   bool
}

// At least 2F + 1 witnesses are needed
// for example F = 1, N = 2 * 1 + 1, 3 witnesses are needed
// ecc will encode data bytes into 3 pieces
// a witnesses need at least 3 - 1 = 2 pieces to recover data
func NewRBC(cfg Config, acs *ACS, groupId, proposerPubkey string) (*RBC, error) {
	rbc_log.Infof("NewRBC called, witnesses pubkey %s, epoch %d", proposerPubkey, acs.epoch)

	parityShards := cfg.F
	if parityShards == 0 {
		parityShards = 1
	}
	dataShards := cfg.N - cfg.F

	// initial reed solomon codec
	enc, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		return nil, err
	}

	rbc := &RBC{
		Config:          cfg,
		acs:             acs,
		groupId:         groupId,
		proposerPubkey:  proposerPubkey,
		enc:             enc,
		recvProofs:      Proofs{},
		recvReadys:      make(map[string]*quorumpb.Ready),
		numParityShards: parityShards,
		numDataShards:   dataShards,
		consenusDone:    false,
	}

	return rbc, nil
}

// when input val in bytes to the rbc instance for myself, the instance will
// 1. seperate bytes to [][]bytes by using reed solomon codec
// 2. make proofReq for each pieces
// 3. broadcast all proofReq via pubsub
func (r *RBC) InputValue(data []byte) error {
	rbc_log.Infof("Input value called, data length %d", len(data))
	//rbc_log.Infof("raw trxBundle %v", data)
	shards, err := makeShards(r.enc, data)
	if err != nil {
		return err
	}

	//create RBC msg for each shards
	reqs, err := r.makeRBCProofMessages(shards)
	if err != nil {
		return err
	}

	rbc_log.Infof("ProofMsg length %d", len(reqs))

	// broadcast RBC msg out via pubsub
	for _, req := range reqs {
		err := SendHbbRBC(r.groupId, req, r.acs.epoch)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *RBC) makeRBCProofMessages(shards [][]byte) ([]*quorumpb.BroadcastMsg, error) {
	rbc_log.Infof("makeRBCProofMessages called")
	msgs := make([]*quorumpb.BroadcastMsg, len(shards))

	for i := 0; i < len(msgs); i++ {
		tree := merkletree.New(sha256.New())
		tree.SetIndex(uint64(i))
		for j := 0; j < len(shards); j++ {
			tree.Push(shards[i])
		}
		root, proof, proofIndex, n := tree.Prove()

		//TBD, call localcypto to sign root_hash with proposerPubkey(mySignPubkey)
		signature := []byte("FAKE_SIGN")

		payload := &quorumpb.Proof{

			RootHash:       root,
			Proof:          proof,
			Index:          int64(proofIndex),
			Leaves:         int64(n),
			ProposerPubkey: r.proposerPubkey,
			ProposerSign:   signature,
		}

		payloadb, err := proto.Marshal(payload)
		if err != nil {
			return nil, err
		}

		msgs[i] = &quorumpb.BroadcastMsg{
			Type:    quorumpb.BroadcastMsgType_PROOF,
			Payload: payloadb,
		}
	}

	return msgs, nil
}

func (r *RBC) makeRBCReadyMessage(proof *quorumpb.Proof) (*quorumpb.BroadcastMsg, error) {
	rbc_log.Infof("makeRBCReadyMessage called")
	//convert group user pubkey to byte

	//sign root_hash with my pubkey
	signature := []byte("FAKE_SIGN")

	ready := &quorumpb.Ready{
		RootHash:            proof.RootHash,
		ProofProviderPubkey: proof.ProposerPubkey, //pubkey for who send the original proof msg
		ProposerPubkey:      r.MySignPubkey,
		ProposerSign:        signature,
	}

	payloadb, err := proto.Marshal(ready)
	if err != nil {
		return nil, err
	}

	readyMsg := &quorumpb.BroadcastMsg{
		Type:    quorumpb.BroadcastMsgType_READY,
		Payload: payloadb,
	}

	return readyMsg, nil
}

func (r *RBC) handleProofMsg(proof *quorumpb.Proof) error {
	rbc_log.Infof("PROOF_MSG:ProofProviderPubkey <%s>, epoch %d", proof.ProposerPubkey, r.acs.epoch)
	if r.consenusDone {
		//rbc done, do nothing, ignore the msg
		rbc_log.Infof("rbc is done, do nothing")
		return nil
	}

	if r.dataDecodeDone {
		rbc_log.Infof("Data decode done, do nothing")
		return nil
	}

	//check proposerPubkey in producer list
	isInProducerList := false
	for _, nodePubkey := range r.Nodes {
		if nodePubkey == string(proof.ProposerPubkey) {
			isInProducerList = true
			break
		}
	}

	if !isInProducerList {
		return fmt.Errorf("receive proof from non producer node <%s>", proof.ProposerPubkey)
	}

	//TBD check signature
	signOk := true
	if !signOk {
		return fmt.Errorf("invalid proof signature")
	}

	if !validateProof(proof) {
		return fmt.Errorf("Received invalid proof from producer node <%s>", proof.ProposerPubkey)
	}

	//save proof
	rbc_log.Infof("Save proof")
	r.recvProofs = append(r.recvProofs, proof)

	//if got enough proof, try decode it
	if r.recvProofs.Len() == r.N-r.F {
		rbc_log.Infof("Try decode")
		err := r.tryDecodeValue()
		if err != nil {
			return err
		}

		rbc_log.Infof("Data is ready")
		r.dataDecodeDone = true

		rbc_log.Infof("broadcast ready msg")
		readyMsg, err := r.makeRBCReadyMessage(proof)
		if err != nil {
			return err
		}

		err = SendHbbRBC(r.groupId, readyMsg, r.acs.epoch)
		if err != nil {
			return err
		}

		//check if we already receive enough readyMsg (N - F)
		rbc_log.Infof("r.recvReadys: %d, r.N-r.F: %d .", len(r.recvReadys), r.N-r.F)
		if len(r.recvReadys) == r.N-r.F {
			rbc_log.Infof("RBC done")
			r.consenusDone = true
			r.acs.RbcDone(r.proposerPubkey)
		} else {
			rbc_log.Infof("wait more ready")
		}
	}

	return nil
}

func (r *RBC) handleReadyMsg(ready *quorumpb.Ready) error {
	rbc_log.Infof("READY_MSG, ProofProviderPubkey <%s>, ProofProposerId <%s>, epoch %d", ready.ProofProviderPubkey, ready.ProposerPubkey, r.acs.epoch)
	if r.consenusDone {
		rbc_log.Infof("Rbc is already done, do nothing")
		return nil
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
		return fmt.Errorf("receive READY from non producer <%s>", ready.ProposerPubkey)
	}

	//check signature with ready.root_hash , ready.Proposer.Pubkey, ready.proposer.Sign
	signOk := true
	if !signOk {
		return fmt.Errorf("invalid ready signature")
	}

	if _, ok := r.recvReadys[string(ready.ProposerPubkey)]; ok {
		return fmt.Errorf("Received multiple readys from <%s>", ready.ProposerPubkey)
	}

	r.recvReadys[string(ready.ProposerPubkey)] = ready

	//check if get enough ready
	if len(r.recvReadys) == r.N-r.F && r.dataDecodeDone {
		rbc_log.Infof("RBC done")
		r.consenusDone = true
		r.acs.RbcDone(r.proposerPubkey)
	} else {
		//wait till enough
		rbc_log.Infof("wait for more READY")
	}

	return nil
}

func validateProof(req *quorumpb.Proof) bool {
	rbc_log.Infof("validateProof called")
	return merkletree.VerifyProof(
		sha256.New(),
		req.RootHash,
		req.Proof,
		uint64(req.Index),
		uint64(req.Leaves))
}

func (r *RBC) tryDecodeValue() error {
	rbc_log.Infof("tryDecodeValue called")
	//sort proof by indexId
	sort.Sort(r.recvProofs)

	shards := make([][]byte, r.numParityShards+r.numDataShards)
	for _, p := range r.recvProofs {
		rbc_log.Infof("index %d", p.Index)
		shards[p.Index] = p.Proof[0]
	}

	if err := r.enc.Reconstruct(shards); err != nil {
		return nil
	}

	var value []byte
	for _, data := range shards[:r.numDataShards] {

		rbc_log.Infof("tryDecodeValue called")
		value = append(value, data...)
	}

	r.output = value

	rbc_log.Infof("Decode done")
	return nil
}

func makeShards(enc reedsolomon.Encoder, data []byte) ([][]byte, error) {
	rbc_log.Infof("makeShards called")
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
