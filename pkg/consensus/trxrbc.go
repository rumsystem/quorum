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
	ecc             reedsolomon.Encoder

	recvProofs Proofs
	recvReadys map[string]*quorumpb.Ready

	output []byte

	readySent    bool
	waitMoreEcho bool
	consenusDone bool
}

// f : maximum failable node
// N : total node
// request : f * 3 < N
// for example
//
//	3 producers node (owner included), 0 * 3 < 3, 0 failable node
//	4 producers node (owner included), 1 * 3 < 4, 1 failable node
//	10 producers node (owner included), 3 * 3 < 10, 3 failable node
//
// ecc will encode data bytes into (N) pieces, each node needs (N - 2f) pieces to recover data
func NewTrxRBC(cfg Config, acs *TrxACS, groupId, proposerPubkey string) (*TrxRBC, error) {
	trx_rbc_log.Infof("NewTrxRBC called, witnesses pubkey %s, epoch %d", proposerPubkey, acs.epoch)

	var (
		parityShards = 2 * cfg.f            //2f
		dataShards   = cfg.N - parityShards //N - 2f
	)

	ds := "vFbwrLArDK"
	data := []byte(ds)

	fmt.Println(data)
	trx_rbc_log.Debugf("%v", data)
	var ecctest reedsolomon.Encoder
	ecctest, _ = reedsolomon.New(1, 0)

	shards, err := ecctest.Split(data)
	if err != nil {
		return nil, err
	}

	if err := ecctest.Encode(shards); err != nil {
		return nil, err
	}

	// initial reed solomon codec
	ecc, err := reedsolomon.New(dataShards, 0) //DataShards N-2f parityShards: 2f , totally N pieces
	if err != nil {
		return nil, err
	}

	trx_rbc_log.Infof("Init reedsolomon codec, datashards <%d>, parityShards<%d>", dataShards, parityShards)

	rbc := &TrxRBC{
		Config:          cfg,
		acs:             acs,
		groupId:         groupId,
		proposerPubkey:  proposerPubkey,
		ecc:             ecc,
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
func (r *TrxRBC) InputValue(data []byte) error {
	trx_rbc_log.Infof("<%s>Input value called, data length %d", r.proposerPubkey, len(data))
	shards, err := MakeShards(r.ecc, data)
	if err != nil {
		return err
	}

	originalDataSize := len(data)
	//create RBC msg for each shards
	reqs, err := MakeRBCProofMessages(r.groupId, r.acs.bft.producer.nodename, r.MySignPubkey, shards, originalDataSize)
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

	/*
		if r.consenusDone {
			//rbc done, do nothing, ignore the msg
			trx_rbc_log.Infof("<%s> rbc is done, do nothing", r.proposerPubkey)
			return nil
		}

		if r.dataDecodeDone {
			trx_rbc_log.Infof("<%s> Data decode done, do nothing", r.proposerPubkey)
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

	//got enough proof
	if r.waitMoreEcho && r.recvProofs.Len() == r.N-2*r.f {
		//already get 2F + 1 ready, try decode data
		trx_rbc_log.Debugf("<%s> try decode", r.proposerPubkey)
		output, err := TryDecodeValue(r.recvProofs, r.ecc, r.numParityShards, r.numDataShards)
		if err != nil {
			return err
		}
		r.output = output

		//let acs know
		trx_rbc_log.Debugf("<%s> rbc is done", r.proposerPubkey)
		r.acs.RbcDone(r.proposerPubkey)
		r.consenusDone = true
	} else if r.recvProofs.Len() == r.N-r.f {
		//check if ready sent
		if r.readySent {
			return nil
		}

		//multicast READY msg
		trx_rbc_log.Debugf("<%s> broadcast ready msg", r.proposerPubkey)
		readyMsg, err := MakeRBCReadyMessage(r.groupId, r.acs.bft.producer.nodename, r.MySignPubkey, proof.RootHash, proof.ProposerPubkey)
		if err != nil {
			return err
		}

		err = SendHbbRBC(r.groupId, readyMsg, r.acs.epoch, quorumpb.HBMsgPayloadType_HB_TRX, "")
		if err != nil {
			return err
		}

		r.readySent = true
	}

	return nil
}

func (r *TrxRBC) handleReadyMsg(ready *quorumpb.Ready) error {
	trx_rbc_log.Debugf("<%s> handle READY_MSG, ProofProviderPubkey <%s>, ReadyMsgProposerId <%s>, epoch <%d>", r.proposerPubkey, ready.ProofProviderPubkey, ready.ProposerPubkey, r.acs.epoch)

	/*
		if r.consenusDone {
			trx_rbc_log.Debugf("<%s> RBC is already done, do nothing", r.proposerPubkey)
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
		return fmt.Errorf("<%s> receive READY from non producer <%s>", r.proposerPubkey, ready.ProposerPubkey)
	}

	//TBD check signature with ready.root_hash , ready.Proposer.Pubkey, ready.proposer.Sign
	signOk := true
	if !signOk {
		return fmt.Errorf("<%s> invalid ready signature", r.proposerPubkey)
	}

	//save it
	r.recvReadys[string(ready.ProposerPubkey)] = ready

	//check if get enough ready
	trx_rbc_log.Debugf("<%s> Recvived ReadyMsg: %d", r.proposerPubkey, len(r.recvReadys))
	trx_rbc_log.Debugf("f %d", r.f)

	if (r.f != 0 && len(r.recvReadys) == 2*r.f+1) || (r.f == 0 && len(r.recvReadys) == r.N) {
		if r.f != 0 {
			trx_rbc_log.Debugf("<%s> get 2f + 1 (%d) READY", r.proposerPubkey, 2*r.f+1)
		} else {
			trx_rbc_log.Debugf("<%s> get N (%d) READY", r.proposerPubkey, r.N)
		}
		if len(r.recvProofs) >= r.N-2*r.f {
			//already receive (N-2f) echo messages, try decode it
			trx_rbc_log.Debugf("<%s> has enough proof, try decode", r.proposerPubkey)
			output, err := TryDecodeValue(r.recvProofs, r.ecc, r.numParityShards, r.numDataShards)
			if err != nil {
				return err
			}
			r.output = output
			//let acs know
			trx_rbc_log.Debugf("<%s> rbc is done", r.proposerPubkey)
			r.acs.RbcDone(r.proposerPubkey)
			r.consenusDone = true
		} else {
			trx_rbc_log.Debugf("<%s> wait for more proof MSG", r.proposerPubkey)
			r.waitMoreEcho = true
		}
	} else if len(r.recvReadys) == r.f+1 {
		if !r.readySent {
			//send ready out
			trx_rbc_log.Debugf("<%s> get f + 1 READY, READY not send,broadcast ready msg", r.proposerPubkey)
			readyMsg, err := MakeRBCReadyMessage(r.groupId, r.acs.bft.producer.nodename, r.MySignPubkey, ready.RootHash, ready.ProposerPubkey)
			if err != nil {
				return err
			}

			err = SendHbbRBC(r.groupId, readyMsg, r.acs.epoch, quorumpb.HBMsgPayloadType_HB_TRX, "")
			if err != nil {
				return err
			}

			r.readySent = true
		}
	} else {
		//wait till get enough READY
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
