package consensus

import (
	"fmt"

	guuid "github.com/google/uuid"
	"github.com/klauspost/reedsolomon"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"google.golang.org/protobuf/proto"

	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var ptrbc_log = logging.Logger("ptrbc")

type PTRbc struct {
	Config
	rbcInstPubkey string

	numParityShards int
	numDataShards   int
	ecc             reedsolomon.Encoder

	recvEchos  map[string]Echos             //key is string(roothash)
	recvReadys map[string][]*quorumpb.Ready //key is string(roothash)

	output []byte

	readySent    map[string]bool
	consenusDone bool

	acs *PTAcs //for callback when finished
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
func NewPTRBC(cfg Config, acs *PTAcs, rbcInstPubkey string) (*PTRbc, error) {
	//ptrbc_log.Infof("NewPTRBC called, epoch <%d> pubkey <%s>", acs.epoch, rbcInstPubkey)

	var (
		parityShards = 2 * cfg.f            //2f
		dataShards   = cfg.N - parityShards //N - 2f
	)

	//initial reed solomon codec
	ecc, err := reedsolomon.New(dataShards, parityShards) //DataShards N-2f parityShards: 2f , totally N pieces
	if err != nil {
		return nil, err
	}

	//ptrbc_log.Infof("Init reedsolomon codec, datashards <%d>, parityShards<%d>", dataShards, parityShards)

	rbc := &PTRbc{
		Config:          cfg,
		acs:             acs,
		rbcInstPubkey:   rbcInstPubkey,
		ecc:             ecc,
		recvEchos:       make(map[string]Echos),
		recvReadys:      make(map[string][]*quorumpb.Ready),
		numParityShards: parityShards,
		numDataShards:   dataShards,
		readySent:       make(map[string]bool),
		consenusDone:    false,
	}

	return rbc, nil
}

// when input val in bytes to the rbc instance for myself, the instance will
// 1. seperate bytes to [][]bytes by using reed solomon codec
// 2. make InitPropose for each nodes
// 3. broadcast all InitPropose
func (r *PTRbc) InputValue(data []byte) error {
	ptrbc_log.Debugf("<%s> Input value called, data length <%d>", r.rbcInstPubkey, len(data))

	//create shards
	shards, err := MakeShards(r.ecc, data)
	if err != nil {
		return err
	}

	//create InitPropoeMsgs
	originalDataSize := len(data)
	initProposeMsgs, err := MakeRBCInitProposeMessage(r.GroupId, r.NodeName, r.MyPubkey, r.MyKeyName, shards, r.Nodes, originalDataSize)

	if err != nil {
		ptrbc_log.Debugf(err.Error())
		return err
	}

	// broadcast RBC msg out via pubsub
	for _, initMsg := range initProposeMsgs {
		r.SendHBRBCMsg(initMsg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *PTRbc) handleInitProposeMsg(initp *quorumpb.InitPropose) error {
	//ptrbc_log.Infof("<%s> handleInitProposeMsg: Proposer <%s>, receiver <%s>, epoch <%d>", r.rbcInstPubkey, initp.ProposerPubkey, initp.RecvNodePubkey, r.acs.Epoch)
	if !r.IsProducer(initp.ProposerPubkey) {
		return fmt.Errorf("<%s> receive proof from non producer <%s>", r.rbcInstPubkey, initp.ProposerPubkey)
	}

	if !r.VerifySign() {
		return fmt.Errorf("<%s> verify signature failed from producer <%s>", r.rbcInstPubkey, initp.ProposerPubkey)
	}

	//valid initP msg
	if isValid := ValidateInitPropose(initp); !isValid {
		return fmt.Errorf("<%s> receive invalid InitPropose msg from producer<%s>", r.rbcInstPubkey, initp.ProposerPubkey)
	}

	//make proof
	proofMsg, err := MakeRBCEchoMessage(r.GroupId, r.NodeName, r.MyPubkey, r.MyKeyName, initp, int(initp.OriginalDataSize))
	if err != nil {
		return err
	}

	//ptrbc_log.Infof("<%s> create and send Echo msg for proposer <%s>", r.rbcInstPubkey, initp.ProposerPubkey)
	return r.SendHBRBCMsg(proofMsg)
}

func (r *PTRbc) handleEchoMsg(echo *quorumpb.Echo) error {
	ptrbc_log.Infof("<%s> handleEchoMsg: EchoProviderPubkey <%s>, epoch <%d>", r.rbcInstPubkey, echo.EchoProviderPubkey, r.acs.epoch)

	if !r.IsProducer(echo.EchoProviderPubkey) {
		return fmt.Errorf("<%s> receive ECHO from non producer node <%s>", r.rbcInstPubkey, echo.EchoProviderPubkey)
	}

	if !r.VerifySign() {
		return fmt.Errorf("<%s> verify ECHO signature failed from producer node <%s>", r.rbcInstPubkey, echo.EchoProviderPubkey)
	}

	if !ValidateEcho(echo) {
		return fmt.Errorf("<%s> received invalid ECHO from producer node <%s>", r.rbcInstPubkey, echo.EchoProviderPubkey)
	}

	roothashS := string(echo.RootHash)
	//save echo by using roothash
	//ptrbc_log.Debugf("<%s> Save ECHO with roothash <%v>", r.rbcInstPubkey, echo.RootHash[:8])

	r.recvEchos[roothashS] = append(r.recvEchos[roothashS], echo)

	//ptrbc_log.Debugf("<%s> RootHash <%v>, Recvived <%d> ECHO", r.rbcInstPubkey, echo.RootHash[:8], r.recvEchos[roothashS].Len())

	if len(r.recvReadys[roothashS]) == 2*r.f+1 && r.recvEchos[roothashS].Len() >= r.N-2*r.f {
		/*
			ptrbc_log.Debugf("<%s> RootHash <%s>, Recvived <%d> READY, which is 2F + 1", r.rbcInstPubkey, roothashS, len(r.recvReadys))
			ptrbc_log.Debugf("<%s> RootHash <%s>, Received <%d> ECHO, which is morn than N - 2F", r.rbcInstPubkey, roothashS, r.recvEchos[roothashS].Len())
			ptrbc_log.Debugf("<%s> RootHash <%s>, try decode", r.rbcInstPubkey)

		*/
		output, err := TryDecodeValue(r.recvEchos[roothashS], r.ecc, r.numParityShards, r.numDataShards)
		if err != nil {
			return err
		}

		//ptrbc_log.Debugf("<%s> RBC for roothash <%s> is done", r.rbcInstPubkey, roothashS)
		r.acs.RbcDone(r.rbcInstPubkey)
		r.consenusDone = true
		r.output = output

		return nil
	}

	/*
		• upon receiving valid ECHO(h,·,·) messages from N − f distinct parties,
		– interpolate {s', j} from any N −2f leaves received
		– recompute Merkle root h0 and if h0 != h then abort
		– if READY(h) has not yet been sent, multicast READY(h)
	*/
	if r.recvEchos[roothashS].Len() == r.N-r.f {
		//ptrbc_log.Debugf("<%s> get N-F echo for rootHash <%v>, try decode", r.rbcInstPubkey, echo.RootHash[:8])
		output, err := TryDecodeValue(r.recvEchos[roothashS], r.ecc, r.numParityShards, r.numDataShards)
		if err != nil {
			return err
		}

		//TBD, recal merkle root hash h0, compare with original root hash

		//check if ready sent
		if r.readySent[roothashS] {
			return nil
		}

		//multicast READY msg
		//ptrbc_log.Debugf("<%s> broadcast READY msg", r.rbcInstPubkey)
		readyMsg, err := MakeRBCReadyMessage(r.GroupId, r.NodeName, r.MyPubkey, r.MyKeyName, echo.OriginalProposerPubkey, echo.RootHash)
		if err != nil {
			return err
		}

		err = r.SendHBRBCMsg(readyMsg)
		if err != nil {
			return err
		}

		//set ready sent
		r.readySent[roothashS] = true

		//set output
		r.output = output
	}

	return nil
}

func (r *PTRbc) handleReadyMsg(ready *quorumpb.Ready) error {
	//ptrbc_log.Debugf("<%s> handle READY_MSG, ReadyProviderPubkey <%s>,  epoch <%d>", r.rbcInstPubkey, ready.ReadyProviderPubkey, r.acs.Epoch)

	if !r.IsProducer(ready.ReadyProviderPubkey) {
		return fmt.Errorf("<%s> receive READY from non producer node <%s>", r.rbcInstPubkey, ready.ReadyProviderPubkey)
	}

	if !r.VerifySign() {
		return fmt.Errorf("<%s> verify READY signature failed from producer node <%s>", r.rbcInstPubkey, ready.ReadyProviderPubkey)
	}

	roothashS := string(ready.RootHash)

	//save it
	//ptrbc_log.Debugf("<%s> Save READY with roothash <%v>", r.rbcInstPubkey, ready.RootHash[:8])
	r.recvReadys[roothashS] = append(r.recvReadys[roothashS], ready)

	//ptrbc_log.Debugf("<%s> RootHash <%v>, Recvived <%d> READY", r.rbcInstPubkey, ready.RootHash[:8], len(r.recvReadys[roothashS]))

	if r.consenusDone {
		//ptrbc_log.Debugf("<%s> RootHash <%v>, RBC is done, do nothing", r.rbcInstPubkey, ready.RootHash[:8])
		return nil
	}

	/*
		upon receiving f +1 matching READY(h) messages, if READY has not yet been sent, multicast READY(h)
	*/

	if len(r.recvReadys[roothashS]) == r.f+1 {
		//ptrbc_log.Debugf("<%s> RootHash <%v>, get f + 1 <%d> READY", r.rbcInstPubkey, ready.RootHash[:8], r.f+1)
		if !r.readySent[roothashS] {
			//ptrbc_log.Debugf("<%s> READY not send, boradcast now", r.rbcInstPubkey)
			readyMsg, err := MakeRBCReadyMessage(r.GroupId, r.NodeName, r.MyPubkey, r.MyKeyName, ready.OriginalProposerPubkey, ready.RootHash)
			if err != nil {
				return err
			}

			err = r.SendHBRBCMsg(readyMsg)
			if err != nil {
				return err
			}

			//set ready sent
			r.readySent[roothashS] = true
		}
	}

	/*
		upon receiving 2 f +1 matching READY(h) messages, wait for (at least) N −2f ECHO messages, then decode v
	*/
	if len(r.recvReadys[roothashS]) >= 2*r.f+1 {
		//ptrbc_log.Debugf("<%s> RootHash <%v>, Recvived <%d> READY, which is more than 2F + 1", r.rbcInstPubkey, ready.RootHash[:8], len(r.recvReadys))
		if r.recvEchos[roothashS].Len() >= r.N-2*r.f {
			//	ptrbc_log.Debugf("<%s> RootHash <%v>, Received <%d> ECHO, which is more than N - 2F", r.rbcInstPubkey, ready.RootHash[:8], r.recvEchos[roothashS].Len())
			if r.output == nil {
				//	ptrbc_log.Debugf("<%s> RootHash <%v>, not decoded yet, try decode", r.rbcInstPubkey, ready.RootHash[:8])
				output, err := TryDecodeValue(r.recvEchos[roothashS], r.ecc, r.numParityShards, r.numDataShards)
				if err != nil {
					return err
				}
				r.output = output
			} else {
				//ptrbc_log.Debugf("<%s> RootHash <%v>, already decoded", r.rbcInstPubkey, ready.RootHash[:8])
			}

			//ptrbc_log.Debugf("<%s> Roothash <%v>, RBC is done", r.rbcInstPubkey, ready.RootHash[:8])
			r.consenusDone = true
			r.acs.RbcDone(r.rbcInstPubkey)

			return nil
		} else {
			//ptrbc_log.Debugf("<%s> RootHash <%v> get enough READY but wait for more ECHO(now has <%d> ECHO)", r.rbcInstPubkey, ready.RootHash[:8], r.recvEchos[roothashS].Len())
			return nil
		}
	}

	//wait till get enough READY
	//ptrbc_log.Debugf("<%s> RootHash <%v> wait for more READY", r.rbcInstPubkey, ready.RootHash[:8])
	return nil
}

func (r *PTRbc) Output() []byte {
	if r.output != nil {
		output := r.output
		r.output = nil
		return output
	}
	return nil
}

func (r *PTRbc) IsProducer(pubkey string) bool {
	for _, nodePubkey := range r.Nodes {
		if nodePubkey == pubkey {
			return true
		}
	}
	return false
}

func (r *PTRbc) VerifySign() bool {
	return true
}

func (r *PTRbc) SendHBRBCMsg(msg *quorumpb.RBCMsg) error {
	ptrbc_log.Debugf("<%s> SendHBRBCMsg called", r.GroupId)
	rbcb, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	hbmsg := &quorumpb.HBMsgv1{
		MsgId:       guuid.New().String(),
		Epoch:       r.acs.epoch,
		ScopeId:     "", //r.acs.consensusInfo.ConsensusId,
		PayloadType: quorumpb.HBMsgPayloadType_RBC,
		Payload:     rbcb,
	}

	hbmsgb, err := proto.Marshal(hbmsg)
	if err != nil {
		return err
	}

	bftMsg := &quorumpb.BftMsg{
		Type: quorumpb.BftMsgType_HB_BFT,
		Data: hbmsgb,
	}

	connMgr, err := conn.GetConn().GetConnMgr(r.GroupId)
	if err != nil {

		return err
	}

	connMgr.BroadcastBftMsg(bftMsg)
	return nil
}
