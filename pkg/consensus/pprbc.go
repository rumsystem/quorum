package consensus

import (
	"fmt"

	"github.com/klauspost/reedsolomon"
	"github.com/rumsystem/quorum/internal/pkg/logging"

	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var pprbc_log = logging.Logger("pprbc")

type PPRbc struct {
	Config

	groupId       string
	myPubkey      string
	rbcInstPubkey string

	numParityShards int
	numDataShards   int

	ecc reedsolomon.Encoder

	recvEchos  map[string]Echos             //key is string(roothash)
	recvReadys map[string][]*quorumpb.Ready //key is string(roothash)

	output []byte

	readySent    map[string]bool
	consenusDone bool

	acs *PPAcs //for callback when finished

	msgSender *HBMsgSender
}

// same as trx rbc
func NewPPRbc(cfg Config, acs *PPAcs, groupId, myPubkey, rbcInstPubkey string) (*PPRbc, error) {
	pprbc_log.Debugf("NewPPRbc called, epoch <%d>  pubkey <%s>", acs.Epoch, rbcInstPubkey)

	var (
		parityShards = 2 * cfg.f            //2f
		dataShards   = cfg.N - parityShards //N - 2f
	)

	if parityShards == 0 {
		parityShards = 1
	}

	//initial reed solomon codec
	ecc, err := reedsolomon.New(dataShards, parityShards) //DataShards N-2f parityShards: 2f , totally N pieces
	if err != nil {
		return nil, err
	}

	rbc := &PPRbc{
		Config:          cfg,
		acs:             acs,
		groupId:         groupId,
		myPubkey:        myPubkey,
		rbcInstPubkey:   rbcInstPubkey,
		ecc:             ecc,
		recvEchos:       make(map[string]Echos),
		recvReadys:      make(map[string][]*quorumpb.Ready),
		numParityShards: parityShards,
		numDataShards:   dataShards,
		readySent:       make(map[string]bool),
		consenusDone:    false,
		msgSender:       NewHBMsgSender(groupId, acs.Epoch, myPubkey, quorumpb.PackageType_HBB_PP),
	}

	return rbc, nil
}

func (r *PPRbc) InputValue(data []byte) error {
	pprbc_log.Debugf("<%s> Input value called, data length <%d>", r.rbcInstPubkey, len(data))

	//create shards
	shards, err := MakeShards(r.ecc, data)
	if err != nil {
		return err
	}

	//create InitPropoeMsgs
	originalDataSize := len(data)
	initProposeMsgs, err := MakeRBCInitProposeMessage(r.groupId, r.acs.bft.pp.nodename, r.MyPubkey, shards, r.Config.Nodes, originalDataSize)

	if err != nil {
		pprbc_log.Debugf(err.Error())
		return err
	}

	// broadcast RBC msg out via pubsub
	for _, initMsg := range initProposeMsgs {
		err := r.msgSender.SendHBRBCMsg(initMsg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *PPRbc) handleInitProposeMsg(initp *quorumpb.InitPropose) error {
	pprbc_log.Infof("<%s> handleInitProposeMsg: Proposer <%s>, receiver <%s>, epoch <%d>", r.rbcInstPubkey, initp.ProposerPubkey, initp.RecvNodePubkey, r.acs.Epoch)
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
	proofMsg, err := MakeRBCEchoMessage(r.groupId, r.acs.bft.pp.nodename, r.MyPubkey, initp, int(initp.OriginalDataSize))
	if err != nil {
		return err
	}

	pprbc_log.Infof("<%s> create and send Echo msg for proposer <%s>", r.rbcInstPubkey, initp.ProposerPubkey)
	return r.msgSender.SendHBRBCMsg(proofMsg)
}

func (r *PPRbc) handleEchoMsg(echo *quorumpb.Echo) error {
	pprbc_log.Infof("<%s> handleEchoMsg: EchoProviderPubkey <%s>, epoch <%d>", r.rbcInstPubkey, echo.EchoProviderPubkey, r.acs.Epoch)

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
	pprbc_log.Debugf("<%s> Save ECHO with roothash <%v>", r.rbcInstPubkey, echo.RootHash[:8])

	r.recvEchos[roothashS] = append(r.recvEchos[roothashS], echo)

	pprbc_log.Debugf("<%s> RootHash <%v>, Recvived <%d> ECHO", r.rbcInstPubkey, echo.RootHash[:8], r.recvEchos[roothashS].Len())

	if len(r.recvReadys[roothashS]) == 2*r.f+1 && r.recvEchos[roothashS].Len() >= r.N-2*r.f {
		pprbc_log.Debugf("<%s> RootHash <%s>, Recvived <%d> READY, which is 2F + 1", r.rbcInstPubkey, roothashS, len(r.recvReadys))
		pprbc_log.Debugf("<%s> RootHash <%s>, Received <%d> ECHO, which is morn than N - 2F", r.rbcInstPubkey, roothashS, r.recvEchos[roothashS].Len())
		pprbc_log.Debugf("<%s> RootHash <%s>, try decode", r.rbcInstPubkey)

		output, err := TryDecodeValue(r.recvEchos[roothashS], r.ecc, r.numParityShards, r.numDataShards)
		if err != nil {
			return err
		}

		pprbc_log.Debugf("<%s> RBC for roothash <%s> is done", r.rbcInstPubkey, roothashS)
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
		pprbc_log.Debugf("<%s> get N-F echo for rootHash <%v>, try decode", r.rbcInstPubkey, echo.RootHash[:8])
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
		pprbc_log.Debugf("<%s> broadcast READY msg", r.rbcInstPubkey)
		readyMsg, err := MakeRBCReadyMessage(r.groupId, r.acs.bft.pp.nodename, r.MyPubkey, echo.OriginalProposerPubkey, echo.RootHash)
		if err != nil {
			return err
		}

		err = r.msgSender.SendHBRBCMsg(readyMsg)
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

func (r *PPRbc) handleReadyMsg(ready *quorumpb.Ready) error {
	pprbc_log.Debugf("<%s> handle READY_MSG, ReadyProviderPubkey <%s>,  epoch <%d>", r.rbcInstPubkey, ready.ReadyProviderPubkey, r.acs.Epoch)

	if !r.IsProducer(ready.ReadyProviderPubkey) {
		return fmt.Errorf("<%s> receive READY from non producer node <%s>", r.rbcInstPubkey, ready.ReadyProviderPubkey)
	}

	if !r.VerifySign() {
		return fmt.Errorf("<%s> verify READY signature failed from producer node <%s>", r.rbcInstPubkey, ready.ReadyProviderPubkey)
	}

	roothashS := string(ready.RootHash)

	//save it
	pprbc_log.Debugf("<%s> Save READY with roothash <%v>", r.rbcInstPubkey, ready.RootHash[:8])
	r.recvReadys[roothashS] = append(r.recvReadys[roothashS], ready)

	pprbc_log.Debugf("<%s> RootHash <%v>, Recvived <%d> READY", r.rbcInstPubkey, ready.RootHash[:8], len(r.recvReadys[roothashS]))

	if r.consenusDone {
		pprbc_log.Debugf("<%s> RootHash <%v>, RBC is done, do nothing", r.rbcInstPubkey, ready.RootHash[:8])
		return nil
	}

	/*
		upon receiving f +1 matching READY(h) messages, if READY has not yet been sent, multicast READY(h)
	*/

	if len(r.recvReadys[roothashS]) == r.f+1 {
		pprbc_log.Debugf("<%s> RootHash <%v>, get f + 1 <%d> READY", r.rbcInstPubkey, ready.RootHash[:8], r.f+1)
		if !r.readySent[roothashS] {
			pprbc_log.Debugf("<%s> READY not send, boradcast now", r.rbcInstPubkey)
			readyMsg, err := MakeRBCReadyMessage(r.groupId, r.acs.bft.pp.nodename, r.myPubkey, ready.OriginalProposerPubkey, ready.RootHash)
			if err != nil {
				return err
			}

			err = r.msgSender.SendHBRBCMsg(readyMsg)
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
		pprbc_log.Debugf("<%s> RootHash <%v>, Recvived <%d> READY, which is more than 2F + 1", r.rbcInstPubkey, ready.RootHash[:8], len(r.recvReadys))
		if r.recvEchos[roothashS].Len() >= r.N-2*r.f {
			pprbc_log.Debugf("<%s> RootHash <%v>, Received <%d> ECHO, which is more than N - 2F", r.rbcInstPubkey, ready.RootHash[:8], r.recvEchos[roothashS].Len())
			if r.output == nil {
				pprbc_log.Debugf("<%s> RootHash <%v>, not decoded yet, try decode", r.rbcInstPubkey, ready.RootHash[:8])
				output, err := TryDecodeValue(r.recvEchos[roothashS], r.ecc, r.numParityShards, r.numDataShards)
				if err != nil {
					return err
				}
				r.output = output
			} else {
				pprbc_log.Debugf("<%s> RootHash <%v>, already decoded", r.rbcInstPubkey, ready.RootHash[:8])
			}

			pprbc_log.Debugf("<%s> Roothash <%v>, RBC is done", r.rbcInstPubkey, ready.RootHash[:8])
			r.consenusDone = true
			r.acs.RbcDone(r.rbcInstPubkey)

			return nil
		} else {
			pprbc_log.Debugf("<%s> RootHash <%v> get enough READY but wait for more ECHO(now has <%d> ECHO)", r.rbcInstPubkey, ready.RootHash[:8], r.recvEchos[roothashS].Len())
			return nil
		}
	}

	//wait till get enough READY
	pprbc_log.Debugf("<%s> RootHash <%v> wait for more READY", r.rbcInstPubkey, ready.RootHash[:8])
	return nil
}

func (r *PPRbc) Output() []byte {
	if r.output != nil {
		output := r.output
		r.output = nil
		return output
	}
	return nil
}

func (r *PPRbc) IsProducer(pubkey string) bool {
	for _, nodePubkey := range r.Nodes {
		if nodePubkey == pubkey {
			return true
		}
	}
	return false
}

func (r *PPRbc) VerifySign() bool {
	return true
}
