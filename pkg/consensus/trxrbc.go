package consensus

import (
	"fmt"

	"github.com/klauspost/reedsolomon"
	"github.com/rumsystem/quorum/internal/pkg/logging"

	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var trx_rbc_log = logging.Logger("trbc")

const TRXS_TOTAL_SIZE int = 900 * 1024

type TrxRBC struct {
	Config

	groupId        string
	proposerPubkey string //proposerPubkey is pubkey for participated witnesses node

	acs *TrxACS //for callback when finished

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
func NewTrxRBC(cfg Config, acs *TrxACS, groupId, proposerPubkey string) (*TrxRBC, error) {
	trx_rbc_log.Infof("NewTrxRBC called, witnesses pubkey %s, epoch %d", proposerPubkey, acs.epoch)

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

	rbc := &TrxRBC{
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
func (r *TrxRBC) InputValue(data []byte) error {
	trx_rbc_log.Infof("Input value called, data length %d", len(data))
	//rbc_log.Infof("raw trxBundle %v", data)
	shards, err := MakeShards(r.enc, data)
	if err != nil {
		return err
	}

	//create RBC msg for each shards
	reqs, err := MakeRBCProofMessages(r.groupId, r.acs.bft.producer.nodename, r.MySignPubkey, shards)
	if err != nil {
		return err
	}

	trx_rbc_log.Infof("ProofMsg length %d", len(reqs))

	// broadcast RBC msg out via pubsub
	for _, req := range reqs {
		err := SendHbbRBC(r.groupId, req, r.acs.epoch, quorumpb.HBMsgPayloadType_HB_TRX, "") //sessionId is used by psync
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *TrxRBC) handleProofMsg(proof *quorumpb.Proof) error {
	trx_rbc_log.Infof("PROOF_MSG:ProofProviderPubkey <%s>, epoch %d", proof.ProposerPubkey, r.acs.epoch)

	if r.consenusDone {
		//rbc done, do nothing, ignore the msg
		trx_rbc_log.Infof("rbc is done, do nothing")
		return nil
	}

	if r.dataDecodeDone {
		trx_rbc_log.Infof("Data decode done, do nothing")
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

	//save proof
	trx_rbc_log.Infof("Save proof")
	r.recvProofs = append(r.recvProofs, proof)

	//if got enough proof, try decode it
	if r.recvProofs.Len() == r.N-r.F {
		trx_rbc_log.Infof("Try decode")
		output, err := TryDecodeValue(r.recvProofs, r.enc, r.numParityShards, r.numDataShards)
		if err != nil {
			return err
		}
		r.output = output

		trx_rbc_log.Infof("Data is ready")
		r.dataDecodeDone = true

		trx_rbc_log.Infof("broadcast ready msg")
		readyMsg, err := MakeRBCReadyMessage(r.groupId, r.acs.bft.producer.nodename, r.MySignPubkey, proof)
		if err != nil {
			return err
		}

		err = SendHbbRBC(r.groupId, readyMsg, r.acs.epoch, quorumpb.HBMsgPayloadType_HB_TRX, "") //sessionId is used by psync
		if err != nil {
			return err
		}

		//check if we already receive enough readyMsg (N - F)
		trx_rbc_log.Infof("r.recvReadys: %d, r.N-r.F: %d .", len(r.recvReadys), r.N-r.F)
		if len(r.recvReadys) == r.N-r.F {
			trx_rbc_log.Infof("RBC done")
			r.consenusDone = true
			r.acs.RbcDone(r.proposerPubkey)
		} else {
			trx_rbc_log.Infof("wait more ready")
		}
	}

	return nil
}

func (r *TrxRBC) handleReadyMsg(ready *quorumpb.Ready) error {
	trx_rbc_log.Infof("READY_MSG, ProofProviderPubkey <%s>, ProofProposerId <%s>, epoch %d", ready.ProofProviderPubkey, ready.ProposerPubkey, r.acs.epoch)

	if r.consenusDone {
		trx_rbc_log.Infof("Rbc is already done, do nothing")
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
		trx_rbc_log.Infof("RBC done")
		r.consenusDone = true
		r.acs.RbcDone(r.proposerPubkey)
	} else {
		//wait till enough
		trx_rbc_log.Infof("wait for more READY")
	}

	return nil
}

func (r *TrxRBC) Output() []byte {
	if r.output != nil {
		output := r.output
		r.output = nil
		return output
	}

	return nil
}
