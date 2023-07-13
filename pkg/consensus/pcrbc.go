package consensus

import (
	"context"
	"fmt"
	"time"

	guuid "github.com/google/uuid"
	"github.com/klauspost/reedsolomon"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"google.golang.org/protobuf/proto"

	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var pcrbc_log = logging.Logger("pcrbc")

type PCRbc struct {
	Config

	rbcInstPubkey string

	numParityShards int
	numDataShards   int

	ecc reedsolomon.Encoder

	recvEchos  map[string]Echos             //key is string(roothash)
	recvReadys map[string][]*quorumpb.Ready //key is string(roothash)

	output []byte

	readySent    map[string]bool
	consenusDone bool

	acs *PCAcs //for callback when finished
}

// same as trx rbc
func NewPCRbc(ctx context.Context, cfg Config, acs *PCAcs, groupId, nodename, myPubkey, rbcInstPubkey string) (*PCRbc, error) {
	//pcrbc_log.Debugf("NewPCRbc called, pubkey <%s>", rbcInstPubkey)

	var (
		parityShards = 2 * cfg.f            //2f
		dataShards   = cfg.N - parityShards //N - 2f
	)

	//initial reed solomon codec
	ecc, err := reedsolomon.New(dataShards, parityShards) //DataShards N-2f parityShards: 2f , totally N pieces
	if err != nil {
		return nil, err
	}

	rbc := &PCRbc{
		Config:          cfg,
		rbcInstPubkey:   rbcInstPubkey,
		ecc:             ecc,
		recvEchos:       make(map[string]Echos),
		recvReadys:      make(map[string][]*quorumpb.Ready),
		numParityShards: parityShards,
		numDataShards:   dataShards,
		readySent:       make(map[string]bool),
		consenusDone:    false,
		acs:             acs,
	}

	return rbc, nil
}

func (r *PCRbc) InputValue(data []byte) error {
	pcrbc_log.Debugf("<%s> Input value called, data length <%d>", r.rbcInstPubkey, len(data))

	//create shards
	shards, err := MakeShards(r.ecc, data)
	if err != nil {
		return err
	}

	//create InitPropoeMsgs
	originalDataSize := len(data)
	initProposeMsgs, err := MakeRBCInitProposeMessage(r.GroupId, r.NodeName, r.MyPubkey, shards, r.Config.Nodes, originalDataSize)

	if err != nil {
		pcrbc_log.Debugf(err.Error())
		return err
	}

	pcrbc_log.Debugf("<%s> create InitProposeMsgs, len <%d>", r.rbcInstPubkey, len(initProposeMsgs))

	// broadcast RBC msg out via pubsub
	for _, initMsg := range initProposeMsgs {
		pcrbc_log.Debugf("<%s> send InitProposeMsgs", r.rbcInstPubkey)
		err := r.SendHBRBCMsg(initMsg)
		if err != nil {
			pcrbc_log.Debugf(err.Error())
			return err
		}
		time.Sleep(1000 * time.Millisecond)
	}

	return nil
}

func (r *PCRbc) handleInitProposeMsg(initp *quorumpb.InitPropose) error {
	pcrbc_log.Infof("<%s> handleInitProposeMsg: Proposer <%s>, receiver <%s> (mypubkey <%s>)", r.rbcInstPubkey, initp.ProposerPubkey, initp.RecvNodePubkey, r.Config.MyPubkey)
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
	proofMsg, err := MakeRBCEchoMessage(r.GroupId, r.NodeName, r.MyPubkey, initp, int(initp.OriginalDataSize))
	if err != nil {
		return err
	}

	pcrbc_log.Infof("<%s> create and send Echo msg for proposer <%s>", r.rbcInstPubkey, initp.ProposerPubkey)
	return r.SendHBRBCMsg(proofMsg)
}

func (r *PCRbc) handleEchoMsg(echo *quorumpb.Echo) error {
	pcrbc_log.Infof("<%s> handleEchoMsg: EchoProviderPubkey <%s>", r.rbcInstPubkey, echo.EchoProviderPubkey)

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
	pcrbc_log.Debugf("<%s> Save ECHO with roothash <%v>", r.rbcInstPubkey, echo.RootHash[:8])

	r.recvEchos[roothashS] = append(r.recvEchos[roothashS], echo)

	//pcrbc_log.Debugf("<%s> RootHash <%v>, Recvived <%d> ECHO", r.rbcInstPubkey, echo.RootHash[:8], r.recvEchos[roothashS].Len())

	if len(r.recvReadys[roothashS]) == 2*r.f+1 && r.recvEchos[roothashS].Len() >= r.N-2*r.f {
		pcrbc_log.Debugf("<%s> RootHash <%s>, Recvived <%d> READY, which is 2F + 1", r.rbcInstPubkey, roothashS, len(r.recvReadys))
		pcrbc_log.Debugf("<%s> RootHash <%s>, Received <%d> ECHO, which is morn than N - 2F", r.rbcInstPubkey, roothashS, r.recvEchos[roothashS].Len())
		pcrbc_log.Debugf("<%s> RootHash <%s>, try decode", r.rbcInstPubkey)

		output, err := TryDecodeValue(r.recvEchos[roothashS], r.ecc, r.numParityShards, r.numDataShards)
		if err != nil {
			return err
		}

		//pcrbc_log.Debugf("<%s> RBC for roothash <%s> is done", r.rbcInstPubkey, roothashS)
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
		pcrbc_log.Debugf("<%s> get N-F echo for rootHash <%v>, try decode", r.rbcInstPubkey, echo.RootHash[:8])
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
		//pcrbc_log.Debugf("<%s> broadcast READY msg", r.rbcInstPubkey)
		readyMsg, err := MakeRBCReadyMessage(r.GroupId, r.NodeName, r.MyPubkey, echo.OriginalProposerPubkey, echo.RootHash)
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

func (r *PCRbc) handleReadyMsg(ready *quorumpb.Ready) error {
	pcrbc_log.Debugf("<%s> handle READY_MSG, ReadyProviderPubkey <%s>", r.rbcInstPubkey, ready.ReadyProviderPubkey)

	if !r.IsProducer(ready.ReadyProviderPubkey) {
		return fmt.Errorf("<%s> receive READY from non producer node <%s>", r.rbcInstPubkey, ready.ReadyProviderPubkey)
	}

	if !r.VerifySign() {
		return fmt.Errorf("<%s> verify READY signature failed from producer node <%s>", r.rbcInstPubkey, ready.ReadyProviderPubkey)
	}

	roothashS := string(ready.RootHash)

	//save it
	pcrbc_log.Debugf("<%s> Save READY with roothash <%v>", r.rbcInstPubkey, ready.RootHash[:8])
	r.recvReadys[roothashS] = append(r.recvReadys[roothashS], ready)

	pcrbc_log.Debugf("<%s> RootHash <%v>, Recvived <%d> READY", r.rbcInstPubkey, ready.RootHash[:8], len(r.recvReadys[roothashS]))

	if r.consenusDone {
		pcrbc_log.Debugf("<%s> RootHash <%v>, RBC is done, do nothing", r.rbcInstPubkey, ready.RootHash[:8])
		return nil
	}

	/*
		upon receiving f +1 matching READY(h) messages, if READY has not yet been sent, multicast READY(h)
	*/

	if len(r.recvReadys[roothashS]) == r.f+1 {
		pcrbc_log.Debugf("<%s> RootHash <%v>, get f + 1 <%d> READY", r.rbcInstPubkey, ready.RootHash[:8], r.f+1)
		if !r.readySent[roothashS] {
			pcrbc_log.Debugf("<%s> READY not send, boradcast now", r.rbcInstPubkey)
			readyMsg, err := MakeRBCReadyMessage(r.GroupId, r.NodeName, r.MyPubkey, ready.OriginalProposerPubkey, ready.RootHash)
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
		pcrbc_log.Debugf("<%s> RootHash <%v>, Recvived <%d> READY, which is more than 2F + 1", r.rbcInstPubkey, ready.RootHash[:8], len(r.recvReadys))
		if r.recvEchos[roothashS].Len() >= r.N-2*r.f {
			pcrbc_log.Debugf("<%s> RootHash <%v>, Received <%d> ECHO, which is more than N - 2F", r.rbcInstPubkey, ready.RootHash[:8], r.recvEchos[roothashS].Len())
			if r.output == nil {
				pcrbc_log.Debugf("<%s> RootHash <%v>, not decoded yet, try decode", r.rbcInstPubkey, ready.RootHash[:8])
				output, err := TryDecodeValue(r.recvEchos[roothashS], r.ecc, r.numParityShards, r.numDataShards)
				if err != nil {
					return err
				}
				r.output = output
			} else {
				pcrbc_log.Debugf("<%s> RootHash <%v>, already decoded", r.rbcInstPubkey, ready.RootHash[:8])
			}

			pcrbc_log.Debugf("<%s> Roothash <%v>, RBC is done", r.rbcInstPubkey, ready.RootHash[:8])
			r.consenusDone = true
			r.acs.RbcDone(r.rbcInstPubkey)

			return nil
		} else {
			pcrbc_log.Debugf("<%s> RootHash <%v> get enough READY but wait for more ECHO(now has <%d> ECHO)", r.rbcInstPubkey, ready.RootHash[:8], r.recvEchos[roothashS].Len())
			return nil
		}
	}

	//wait till get enough READY
	pcrbc_log.Debugf("<%s> RootHash <%v> wait for more READY", r.rbcInstPubkey, ready.RootHash[:8])
	return nil
}

func (r *PCRbc) Output() []byte {
	if r.output != nil {
		output := r.output
		r.output = nil
		return output
	}
	return nil
}

func (r *PCRbc) IsProducer(pubkey string) bool {
	for _, nodePubkey := range r.Nodes {
		if nodePubkey == pubkey {
			return true
		}
	}
	return false
}

func (r *PCRbc) VerifySign() bool {
	return true
}

func (r *PCRbc) SendHBRBCMsg(msg *quorumpb.RBCMsg) error {
	rbcb, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	hbmsg := &quorumpb.HBMsgv1{
		MsgId:       guuid.New().String(),
		ScopeId:     r.acs.scopeId,
		Epoch:       r.acs.round,
		PayloadType: quorumpb.HBMsgPayloadType_RBC,
		Payload:     rbcb,
	}

	//marshall hbmsg to bytes
	hbmsgb, err := proto.Marshal(hbmsg)
	if err != nil {
		return err
	}

	//build ccMsg
	ccMsg := &quorumpb.CCMsg{
		Type: quorumpb.CCMsgType_CC_PROOF_HB,
		Data: hbmsgb,
	}

	connMgr, err := conn.GetConn().GetConnMgr(r.GroupId)
	if err != nil {
		return err
	}

	connMgr.BroadcastCCMsg(ccMsg)
	return nil
}
