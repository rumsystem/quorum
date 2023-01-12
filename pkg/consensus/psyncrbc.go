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

	output []byte

	//dataDecodeDone bool
	//onsenusDone   bool

	readySent    bool
	waitMoreEcho bool
	consenusDone bool
}

// same as trx rbc
func NewPSyncRBC(cfg Config, acs *PSyncACS, groupId, proposerPubkey string) (*PSyncRBC, error) {
	psync_rbc_log.Debugf("SessionId <%s> NewPSyncRBC called, witnesses pubkey <%s>", acs.SessionId, proposerPubkey)

	if cfg.f == 0 {
		cfg.f = (cfg.N - 1) / 3
	}
	var (
		parityShards = 2 * cfg.f            //2f
		dataShards   = cfg.N - parityShards //N - 2f
	)

	if parityShards == 0 {
		parityShards = 1
	}

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
		readySent:       false,
		waitMoreEcho:    false,
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
	originalDataSize := len(data)
	reqs, err := MakeRBCProofMessages(r.groupId, r.acs.bft.PSyncer.nodename, r.MySignPubkey, shards, originalDataSize)
	if err != nil {
		return err
	}

	trx_rbc_log.Infof("<%s> ProofMsg length %d", r.proposerPubkey, len(reqs))

	// broadcast RBC msg out via pubsub
	for _, req := range reqs {
		err := SendHbbRBC(r.groupId, req, r.acs.bft.PSyncer.cIface.GetCurrEpoch(), quorumpb.HBMsgPayloadType_HB_PSYNC, r.acs.SessionId)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *PSyncRBC) handleProofMsg(proof *quorumpb.Proof) error {
	psync_rbc_log.Debugf("<%s> SessionId <%s> PROOF_MSG:ProofProviderPubkey <%s>", r.proposerPubkey, r.acs.SessionId, proof.ProposerPubkey)

	/*
		if r.consenusDone {
			//rbc done, do nothing, ignore the msg
			psync_rbc_log.Debugf("<%s> rbc is done, do nothing", r.proposerPubkey)
			return nil
		}

		if r.dataDecodeDone {
			psync_rbc_log.Debugf("<%s> Data decode done, do nothing", r.proposerPubkey)
			return nil
		}
	*/

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

	psync_rbc_log.Debugf("<%s> Save proof", r.proposerPubkey)
	r.recvProofs = append(r.recvProofs, proof)

	//got enough proof
	if r.waitMoreEcho && r.recvProofs.Len() == r.N-2*r.f {
		//already get 2F + 1 ready, try decode data
		psync_rbc_log.Debugf("<%s> try decode", r.proposerPubkey)
		output, err := TryDecodeValue(r.recvProofs, r.enc, r.numParityShards, r.numDataShards)
		if err != nil {
			return err
		}
		r.output = output

		//let acs know
		psync_rbc_log.Debugf("<%s> rbc is done", r.proposerPubkey)
		r.acs.RbcDone(r.proposerPubkey)
		r.consenusDone = true
	} else if r.recvProofs.Len() == r.N-r.f {
		//check if ready sent
		if r.readySent {
			return nil
		}

		//multicast READY msg
		psync_rbc_log.Debugf("<%s> broadcast ready msg", r.proposerPubkey)
		readyMsg, err := MakeRBCReadyMessage(r.groupId, r.acs.bft.PSyncer.nodename, r.MySignPubkey, proof.RootHash, proof.ProposerPubkey)
		if err != nil {
			return err
		}

		err = SendHbbRBC(r.groupId, readyMsg, r.acs.bft.PSyncer.cIface.GetCurrEpoch(), quorumpb.HBMsgPayloadType_HB_PSYNC, r.acs.SessionId)
		if err != nil {
			return err
		}

		r.readySent = true
	}

	return nil
}

func (r *PSyncRBC) handleReadyMsg(ready *quorumpb.Ready) error {
	psync_rbc_log.Debugf("<%s> SessionId <%s> READY_MSG, ProofProviderPubkey <%s>, ProofProposerId <%s>", r.proposerPubkey, r.acs.SessionId, ready.ProofProviderPubkey, ready.ProposerPubkey)

	/*
		if r.consenusDone {
			psync_rbc_log.Debugf("<%s> RBC is already done, do nothing", r.proposerPubkey)
			return nil
		}
	*/

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

	//save it
	r.recvReadys[string(ready.ProposerPubkey)] = ready

	//check if get enough ready
	if len(r.recvReadys) == 2*r.f+1 {
		psync_rbc_log.Debugf("<%s> get 2f + 1 READY", r.proposerPubkey)
		if len(r.recvProofs) >= r.N-2*r.f {
			//already receive (N-2f) echo messages, try decode it
			psync_rbc_log.Debugf("<%s> has enough proof, try decode", r.proposerPubkey)
			output, err := TryDecodeValue(r.recvProofs, r.enc, r.numParityShards, r.numDataShards)
			if err != nil {
				return err
			}
			r.output = output
			//let acs know
			psync_rbc_log.Debugf("<%s> rbc is done", r.proposerPubkey)
			r.acs.RbcDone(r.proposerPubkey)
			r.consenusDone = true
		} else {
			psync_rbc_log.Debugf("<%s> wait for more proof MSG", r.proposerPubkey)
			r.waitMoreEcho = true
		}
	} else if len(r.recvReadys) == r.f+1 {
		if !r.readySent {
			//send ready out
			psync_rbc_log.Debugf("<%s> get f + 1 READY, READY not send,broadcast ready msg", r.proposerPubkey)
			readyMsg, err := MakeRBCReadyMessage(r.groupId, r.acs.bft.PSyncer.nodename, r.MySignPubkey, ready.RootHash, ready.ProposerPubkey)
			if err != nil {
				return err
			}

			err = SendHbbRBC(r.groupId, readyMsg, r.acs.bft.PSyncer.cIface.GetCurrEpoch(), quorumpb.HBMsgPayloadType_HB_PSYNC, r.acs.SessionId)
			if err != nil {
				return err
			}

			r.readySent = true
		}
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
