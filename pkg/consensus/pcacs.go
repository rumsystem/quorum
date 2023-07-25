package consensus

import (
	"context"
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var pcacs_log = logging.Logger("pcacs")

type PCAcs struct {
	Config
	rbcInsts   map[string]*PCRbc
	rbcOutput  map[string]bool
	rbcResults map[string][]byte

	chAcsDone chan *AcsResult
	round     uint64
	scopeId   string
}

func NewPCAcs(ctx context.Context, cfg Config, round uint64, scopeId string, chAcsDone chan *AcsResult) *PCAcs {
	pcacs_log.Debugf("NewPCAcs called, round <%d>", round)

	acs := &PCAcs{
		Config:     cfg,
		round:      round,
		scopeId:    scopeId,
		rbcInsts:   make(map[string]*PCRbc),
		rbcOutput:  make(map[string]bool),
		rbcResults: make(map[string][]byte),

		chAcsDone: chAcsDone,
	}

	for _, nodePubkey := range cfg.Nodes {
		acs.rbcInsts[nodePubkey], _ = NewPCRbc(ctx, cfg, acs, acs.GroupId, acs.NodeName, acs.MyPubkey, nodePubkey)
	}

	return acs
}

// give input value to
func (a *PCAcs) InputValue(val []byte) error {
	pcacs_log.Debug("InputValue called")
	rbc, ok := a.rbcInsts[a.MyPubkey]
	if !ok {
		return fmt.Errorf("could not find rbc instance <%s>", a.MyPubkey)
	}

	return rbc.InputValue(val)
}

// rbc for proposerIs finished
func (a *PCAcs) RbcDone(proposerPubkey string) {
	//pcacs_log.Debugf("RbcDone called, RBC <%s> finished", proposerPubkey)
	pcacs_log.Debugf("Rbc <%s> finished", proposerPubkey)
	a.rbcOutput[proposerPubkey] = true

	if len(a.rbcOutput) == a.N-a.f {
		ptacs_log.Debugf("<%d> RBC finished, BFT done", a.N-a.f)
		for rbcInst, _ := range a.rbcOutput {
			a.rbcResults[rbcInst] = a.rbcInsts[rbcInst].Output()
		}

		//notify acs done
		a.chAcsDone <- &AcsResult{
			result: a.rbcResults,
		}
	}
}

func (a *PCAcs) HandleHBMessage(hbmsg *quorumpb.HBMsgv1) error {
	ptacs_log.Debugf("ACS HandleHBMessage called, acs round <%d>, msgType <%s>", a.round, hbmsg.PayloadType.String())

	//check epoch(round) and scopeId (reqId)
	if hbmsg.Epoch != a.round {
		ptacs_log.Debugf("received HB msg epoch <%d> not match with acs epoch <%d>", hbmsg.Epoch, a.round)
		return fmt.Errorf("received HB msg epoch <%d> not match with acs epoch <%d>", hbmsg.Epoch, a.round)
	}

	if hbmsg.ScopeId != a.scopeId {
		ptacs_log.Debugf("received HB msg scopeId <%s> not match with acs scopeId <%s>", hbmsg.ScopeId, a.scopeId)
		return fmt.Errorf("received HB msg scopeId <%s> not match with acs scopeId <%s>", hbmsg.ScopeId, a.scopeId)
	}

	switch hbmsg.PayloadType {
	case quorumpb.HBMsgPayloadType_RBC:
		return a.handleRbcMsg(hbmsg.Payload)
		//	case quorumpb.HBMsgPayloadType_BBA:
		//		return a.handleBbaMsg(hbmsg.Payload)
	default:
		return fmt.Errorf("received unknown type msg <%s>", hbmsg.PayloadType.String())
	}
}

func (a *PCAcs) handleRbcMsg(payload []byte) error {
	//ptacs_log.Debugf("handleRbcMsg called, Epoch <%d>", a.Epoch)
	//cast payload to RBC message
	rbcMsg := &quorumpb.RBCMsg{}
	err := proto.Unmarshal(payload, rbcMsg)
	if err != nil {
		return err
	}

	switch rbcMsg.Type {
	case quorumpb.RBCMsgType_INIT_PROPOSE:
		initp := &quorumpb.InitPropose{}
		err := proto.Unmarshal(rbcMsg.Payload, initp)
		if err != nil {
			return err
		}

		if initp.RecvNodePubkey != a.MyPubkey {
			ptacs_log.Debugf("INIT_PROPOSE: sender <%s> receiver <%s>, NOT FOR ME, IGNORE", initp.ProposerPubkey, initp.RecvNodePubkey)
			return nil
		}

		rbc, ok := a.rbcInsts[initp.ProposerPubkey]
		if !ok {
			return fmt.Errorf("could not find rbc instance to handle InitPropose form <%s>", initp.ProposerPubkey)
		}
		//ptacs_log.Debugf("INIT_PROPOSE: is for me, handle it")
		return rbc.handleInitProposeMsg(initp)
	case quorumpb.RBCMsgType_ECHO:
		echo := &quorumpb.Echo{}
		err := proto.Unmarshal(rbcMsg.Payload, echo)
		if err != nil {
			return err
		}
		//give the ECHO msg to original proposer
		rbc, ok := a.rbcInsts[echo.OriginalProposerPubkey]
		if !ok {
			return fmt.Errorf("could not find rbc instance to handle proof from <%s>, original propose <%s>", echo.EchoProviderPubkey, echo.OriginalProposerPubkey)
		}
		return rbc.handleEchoMsg(echo)
	case quorumpb.RBCMsgType_READY:
		ready := &quorumpb.Ready{}
		err := proto.Unmarshal(rbcMsg.Payload, ready)
		if err != nil {
			return err
		}
		rbc, ok := a.rbcInsts[ready.OriginalProposerPubkey]
		if !ok {
			return fmt.Errorf("could not find rbc instance to handle ready from <%s>", ready.ReadyProviderPubkey)
		}
		return rbc.handleReadyMsg(ready)

	default:
		return fmt.Errorf("received unknown rbc message, type (%s)", rbcMsg.Type)
	}
}
