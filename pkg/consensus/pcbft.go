package consensus

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var pcbft_log = logging.Logger("pcbft")

type AcsResult struct {
	epoch  uint64
	result map[string][]byte
}

type BftResult struct {
	Result  quorumpb.ChangeConsensusResult
	RawData map[string][]byte
	Epoch   uint64
}

type PCBft struct {
	Config
	groupId  string
	nodename string

	currProof     *quorumpb.ConsensusProof
	currProotData []byte

	agreementEpochLenInMs uint64
	agreementTotalEpoch   uint64

	chBftDone chan *BftResult
	cpCtx     context.Context

	epoch   uint64
	acsInst *PPAcs

	//responsedProducers map[string]bool
}

func NewPCBft(ctx context.Context, groupId, nodename string, cfg Config, agrmEpochLen, agrmTotalEpoch uint64, ch chan *BftResult) *PCBft {
	pcbft_log.Debugf("NewPCBft called")
	return &PCBft{
		Config:                cfg,
		groupId:               groupId,
		nodename:              nodename,
		currProof:             nil,
		epoch:                 0,
		agreementEpochLenInMs: agrmEpochLen,
		agreementTotalEpoch:   agrmTotalEpoch,
		cpCtx:                 ctx,
		chBftDone:             ch,
	}
}

func (bft *PCBft) AddProof(proof *quorumpb.ConsensusProof) {
	pcbft_log.Debugf("AddProducerProposal called, reqid <%s> ", proof.Req.ReqId)
	bft.currProof = proof
	datab, _ := proto.Marshal(proof)
	bft.currProotData = datab
}

func (bft *PCBft) Propose() error {
	pcbft_log.Debugf("Propose called")
	chAcsDone := make(chan *AcsResult, 1)

	go func() {
		acs := NewPPAcs(bft.groupId, bft.nodename, bft.Config, bft.epoch, chAcsDone)
		acs.InputValue(bft.currProotData)
	}()

	for {
		select {
		case <-bft.cpCtx.Done():
			pcbft_log.Debugf("<%s> ctx done", bft.groupId)
			return nil
		case acsResult := <-chAcsDone:
			pcbft_log.Debugf("acs done, epoch <%d>", acsResult.epoch)
			//bft.chBftDone <- bft.makeResultBundle(acsResult.epoch, acsResult.result)
			bft.chBftDone <- &BftResult{
				Result:  quorumpb.ChangeConsensusResult_SUCCESS,
				Epoch:   acsResult.epoch,
				RawData: acsResult.result,
			}
			return nil
		case <-time.After(time.Duration(bft.agreementEpochLenInMs) * time.Millisecond):
			pcbft_log.Debugf("acs <%d> timeout", bft.epoch)
			bft.epoch += 1
			if bft.epoch > uint64(bft.agreementTotalEpoch) {
				pcbft_log.Debugf("bft timeout, could not make agreement")
				bft.chBftDone <- &BftResult{
					Result:  quorumpb.ChangeConsensusResult_TIMEOUT,
					Epoch:   bft.epoch,
					RawData: nil,
				}
				return nil
			}
			bft.Propose()
		}
	}
	return nil
}

func (bft *PCBft) HandleHBMsg(hbmsg *quorumpb.HBMsgv1) error {
	pcbft_log.Debugf("HandleHBMsg called, Epoch <%d>", hbmsg.Epoch)
	if bft.acsInst != nil {
		bft.acsInst.HandleHBMessage(hbmsg)
	}

	return nil
}
