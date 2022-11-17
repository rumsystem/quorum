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
	trx_rbc_log.Infof("<%s>Input value called, data length %d", r.proposerPubkey, len(data))
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

	trx_rbc_log.Infof("<%s> ProofMsg length %d", r.proposerPubkey, len(reqs))

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
	trx_rbc_log.Infof("<%s> handle PROOF_MSG: ProofProviderPubkey <%s>, epoch <%d>", r.proposerPubkey, proof.ProposerPubkey, r.acs.epoch)

	if r.consenusDone {
		//rbc done, do nothing, ignore the msg
		trx_rbc_log.Infof("<%s> rbc is done, do nothing", r.proposerPubkey)
		return nil
	}

	if r.dataDecodeDone {
		trx_rbc_log.Infof("<%s> Data decode done, do nothing", r.proposerPubkey)
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
		return fmt.Errorf("<%s> receive proof from non producer node <%s>", r.proposerPubkey, proof.ProposerPubkey)
	}

	//TBD check signature
	signOk := true
	if !signOk {
		return fmt.Errorf("invalid proof signature")
	}

	if !ValidateProof(proof) {
		return fmt.Errorf("<%s> received invalid proof from producer node <%s>", r.proposerPubkey, proof.ProposerPubkey)
	}

	//save proof
	trx_rbc_log.Debugf("<%s> Save proof", r.proposerPubkey)
	r.recvProofs = append(r.recvProofs, proof)

	//if got enough proof, try decode it
	if r.recvProofs.Len() == r.N-r.F {
		trx_rbc_log.Debugf("<%s> try decode", r.proposerPubkey)
		output, err := TryDecodeValue(r.recvProofs, r.enc, r.numParityShards, r.numDataShards)
		if err != nil {
			return err
		}
		r.output = output

		trx_rbc_log.Debugf("<%s> data is ready", r.proposerPubkey)
		r.dataDecodeDone = true

		trx_rbc_log.Debugf("<%s> broadcast ready msg", r.proposerPubkey)
		readyMsg, err := MakeRBCReadyMessage(r.groupId, r.acs.bft.producer.nodename, r.MySignPubkey, proof)
		if err != nil {
			return err
		}

		err = SendHbbRBC(r.groupId, readyMsg, r.acs.epoch, quorumpb.HBMsgPayloadType_HB_TRX, "")
		if err != nil {
			return err
		}

		//check if we already receive enough readyMsg (N - F)
		trx_rbc_log.Debugf("<%s> recvived ReadyMsg: %d, expected ReadyMsg(r.N-r.F): %d .", r.proposerPubkey, len(r.recvReadys), r.N-r.F)
		if len(r.recvReadys) == r.N-r.F {
			trx_rbc_log.Debugf("<%s> RBC done", r.proposerPubkey)
			r.consenusDone = true
			r.acs.RbcDone(r.proposerPubkey)
		} else {
			trx_rbc_log.Debugf("<%s> wait more ready", r.proposerPubkey)
		}
	}

	return nil
}

func (r *TrxRBC) handleReadyMsg(ready *quorumpb.Ready) error {
	trx_rbc_log.Debugf("<%s> handle READY_MSG, ProofProviderPubkey <%s>, ReadyMsgProposerId <%s>, epoch <%d>", r.proposerPubkey, ready.ProofProviderPubkey, ready.ProposerPubkey, r.acs.epoch)

	if r.consenusDone {
		trx_rbc_log.Debugf("<%s> RBC is already done, do nothing", r.proposerPubkey)
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
		return fmt.Errorf("<%s> receive READY from non producer <%s>", r.proposerPubkey, ready.ProposerPubkey)
	}

	//TBD check signature with ready.root_hash , ready.Proposer.Pubkey, ready.proposer.Sign
	signOk := true
	if !signOk {
		return fmt.Errorf("<%s> invalid ready signature", r.proposerPubkey)
	}

	if _, ok := r.recvReadys[string(ready.ProposerPubkey)]; ok {
		return fmt.Errorf("<%s> received multiple readys from <%s>", r.proposerPubkey, ready.ProposerPubkey)
	}

	r.recvReadys[string(ready.ProposerPubkey)] = ready

	//check if get enough ready
	trx_rbc_log.Debugf("<%s> Recvived ReadyMsg: %d, Expected ReadyMsg(r.N-r.F): %d .", r.proposerPubkey, len(r.recvReadys), r.N-r.F)
	if len(r.recvReadys) == r.N-r.F && r.dataDecodeDone {
		trx_rbc_log.Debugf("<%s> get ENOUGH READY_MSG, RBC done", r.proposerPubkey)
		r.consenusDone = true
		r.acs.RbcDone(r.proposerPubkey)
	} else {
		//wait till enough
		trx_rbc_log.Debugf("<%s> wait for more READY_MSG", r.proposerPubkey)
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
