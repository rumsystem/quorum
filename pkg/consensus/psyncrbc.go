package consensus

import (
	"fmt"

	"github.com/klauspost/reedsolomon"
	"github.com/rumsystem/quorum/internal/pkg/logging"

	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var psync_rbc_log = logging.Logger("prbc")

type PSyncRBC struct {
	Config

	groupId        string
	proposerPubkey string //proposerPubkey is pubkey for participated witnesses node

	acs *PSyncACS //for callback when finished

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
func NewPSyncRBC(cfg Config, acs *PSyncACS, groupId, proposerPubkey string) (*PSyncRBC, error) {
	psync_rbc_log.Debugf("SessionId <%s> NewPSyncRBC called, witnesses pubkey <%s>", acs.SessionId, proposerPubkey)

	parityShards := cfg.f
	if parityShards == 0 {
		parityShards = 1
	}

	dataShards := cfg.N - cfg.f

	// initial reed solomon codec
	enc, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		return nil, err
	}

	rbc := &PSyncRBC{
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
func (r *PSyncRBC) InputValue(data []byte) error {
	psync_rbc_log.Debugf("<%s> SessionId <%s> InputValue called, data length %d", r.proposerPubkey, r.acs.SessionId, len(data))

	shards, err := MakeShards(r.enc, data)
	if err != nil {
		return err
	}

	//create RBC msg for each shards
	reqs, err := MakeRBCProofMessages(r.groupId, r.acs.bft.PSyncer.nodename, r.MySignPubkey, shards)
	if err != nil {
		return err
	}

	// broadcast RBC msg out via pubsub
	for _, req := range reqs {
		err := SendHbbRBC(r.groupId, req, r.acs.bft.PSyncer.grpItem.Epoch, quorumpb.HBMsgPayloadType_HB_PSYNC, r.acs.SessionId)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *PSyncRBC) handleProofMsg(proof *quorumpb.Proof) error {
	psync_rbc_log.Debugf("<%s> SessionId <%s> PROOF_MSG:ProofProviderPubkey <%s>", r.proposerPubkey, r.acs.SessionId, proof.ProposerPubkey)

	if r.consenusDone {
		//rbc done, do nothing, ignore the msg
		psync_rbc_log.Debugf("<%s> rbc is done, do nothing", r.proposerPubkey)
		return nil
	}

	if r.dataDecodeDone {
		psync_rbc_log.Debugf("<%s> Data decode done, do nothing", r.proposerPubkey)
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

	if !ValidateProof(proof) {
		return fmt.Errorf("received invalid proof from producer node <%s>", proof.ProposerPubkey)
	}

	r.recvProofs = append(r.recvProofs, proof)

	//if got enough proof, try decode it
	if r.recvProofs.Len() == r.N-r.f {
		output, err := TryDecodeValue(r.recvProofs, r.enc, r.numParityShards, r.numDataShards)
		if err != nil {
			return err
		}
		r.output = output

		psync_rbc_log.Debugf("<%s> Data is ready", r.proposerPubkey)
		r.dataDecodeDone = true

		psync_rbc_log.Debugf("<%s> broadcast READY msg", r.proposerPubkey)
		readyMsg, err := MakeRBCReadyMessage(r.groupId, r.acs.bft.PSyncer.nodename, r.MySignPubkey, proof.RootHash, r.proposerPubkey)
		if err != nil {
			return err
		}

		err = SendHbbRBC(r.groupId, readyMsg, r.acs.bft.PSyncer.grpItem.Epoch, quorumpb.HBMsgPayloadType_HB_PSYNC, r.acs.SessionId)
		if err != nil {
			return err
		}

		//check if we already receive enough readyMsg (N - F)
		psync_rbc_log.Debugf("<%s> received readyMsg <%d>, need <%d>", r.proposerPubkey, len(r.recvReadys), r.N-r.f)
		if len(r.recvReadys) == r.N-r.f {
			psync_rbc_log.Debugf("<%s> RBC done", r.proposerPubkey)
			r.consenusDone = true
			r.acs.RbcDone(r.proposerPubkey)
		} else {
			psync_rbc_log.Infof("<%s> wait for more READY", r.proposerPubkey)
		}
	}

	return nil
}

func (r *PSyncRBC) handleReadyMsg(ready *quorumpb.Ready) error {
	psync_rbc_log.Debugf("<%s> SessionId <%s> READY_MSG, ProofProviderPubkey <%s>, ProofProposerId <%s>", r.proposerPubkey, r.acs.SessionId, ready.ProofProviderPubkey, ready.ProposerPubkey)

	if r.consenusDone {
		psync_rbc_log.Debugf("<%s> RBC is already done, do nothing", r.proposerPubkey)
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
		return fmt.Errorf("received multiple readys from <%s>", ready.ProposerPubkey)
	}

	r.recvReadys[string(ready.ProposerPubkey)] = ready

	//check if get enough ready
	psync_rbc_log.Debugf("<%s> received readyMsg <%d>, need <%d>", r.proposerPubkey, len(r.recvReadys), r.N-r.f)
	if len(r.recvReadys) == r.N-r.f && r.dataDecodeDone {
		psync_rbc_log.Debugf("<%s> RBC done", r.proposerPubkey)
		r.consenusDone = true
		r.acs.RbcDone(r.proposerPubkey)
	} else {
		//wait till enough
		psync_rbc_log.Debugf("<%s> wait for more READY", r.proposerPubkey)
	}

	return nil
}

func (r *PSyncRBC) Output() []byte {
	if r.output != nil {
		output := r.output
		r.output = nil
		return output
	}

	return nil
}
