package consensus

import (
	"bytes"
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var pcbft_log = logging.Logger("pcbft")

type AcsResult struct {
	result map[string][]byte
}

type PCBft struct {
	Config

	currProof     *quorumpb.ConsensusProof
	currProotData []byte

	chBftDone chan *quorumpb.ChangeConsensusResultBundle
	bftCtx    context.Context

	acsInst *PCAcs

	cIface def.ChainMolassesIface
}

func NewPCBft(ctx context.Context, cfg Config, ch chan *quorumpb.ChangeConsensusResultBundle, iface def.ChainMolassesIface) *PCBft {
	//pcbft_log.Debugf("NewPCBft called")
	return &PCBft{
		Config:    cfg,
		currProof: nil,
		bftCtx:    ctx,
		chBftDone: ch,
		cIface:    iface,
	}
}

func (bft *PCBft) AddProof(proof *quorumpb.ConsensusProof) {
	pcbft_log.Debugf("AddProof called, reqid <%s>, nonce <%d> ", proof.Req.ReqId, proof.Req.Nonce)
	bft.currProof = proof
	datab, _ := proto.Marshal(proof)
	bft.currProotData = datab
}

func (bft *PCBft) Propose() error {
	pcbft_log.Debugf("Propose called")
	chAcsDone := make(chan *AcsResult, 1)

	acs := NewPCAcs(bft.bftCtx, bft.Config, bft.currProof.Epoch, chAcsDone)
	acs.InputValue(bft.currProotData)
	bft.acsInst = acs

	for {
		select {
		case <-bft.bftCtx.Done():
			pcbft_log.Debugf("<%s> bft ctx done, quit peacefully", bft.GroupId)
			return nil
		case acsResult := <-chAcsDone:
			pcbft_log.Debugf("acs done")
			//verify raw result
			ok, proofMap := bft.verifyRawResult(acsResult.result)
			if !ok {
				pcbft_log.Errorf("<%s> verify raw result failed", bft.GroupId)
				resultBundle := &quorumpb.ChangeConsensusResultBundle{
					Result:             quorumpb.ChangeConsensusResult_FAIL,
					Req:                bft.currProof.Req,
					Resps:              nil,
					ResponsedProducers: nil,
				}

				//notify bft done
				bft.chBftDone <- resultBundle
				return fmt.Errorf("bft done but verify raw result failed, quit percefully")
			}

			var resps []*quorumpb.ChangeConsensusResp
			var producers []string
			for _, proof := range proofMap {
				resps = append(resps, proof.Resp)
				producers = append(producers, proof.Resp.SenderPubkey)
			}

			resultBundle := &quorumpb.ChangeConsensusResultBundle{
				Result:             quorumpb.ChangeConsensusResult_SUCCESS,
				Req:                bft.currProof.Req,
				Resps:              resps,
				ResponsedProducers: producers,
			}
			//notify bft done
			bft.chBftDone <- resultBundle
			pcbft_log.Debugf("<%s> bft done, quit peacefully", bft.GroupId)

			return nil
		}
	}
}

func (bft *PCBft) HandleHBMsg(hbmsg *quorumpb.HBMsgv1) error {
	//pcbft_log.Debugf("HandleHBMsg called")
	if bft.acsInst != nil {
		bft.acsInst.HandleHBMessage(hbmsg)
	}
	return nil
}

func (bft *PCBft) verifyRawResult(rawResult map[string][]byte) (bool, map[string]*quorumpb.ConsensusProof) {
	pcbft_log.Debugf("<%s> verifyRawResult called", bft.GroupId)

	//convert rawResultMap to proof map
	proofMap := make(map[string]*quorumpb.ConsensusProof)
	for k, v := range rawResult {
		proof := &quorumpb.ConsensusProof{}
		err := proto.Unmarshal(v, proof)
		if err != nil {
			pcbft_log.Errorf("<%s> unmarshal consensus proof failed", bft.GroupId)
			return false, proofMap
		}
		proofMap[k] = proof
	}

	//check if the maps contains responses from all required producers
	for _, pubkey := range bft.currProof.Req.ProducerPubkeyList {
		proof, ok := proofMap[pubkey]
		if !ok {
			pcbft_log.Errorf("<%s> proof map does not contains producerPubkey <%s>", bft.GroupId, pubkey)
			return false, proofMap
		}

		//check if sender is as same as producerPubkey
		if proof.Resp.SenderPubkey != pubkey {
			pcbft_log.Errorf("<%s> proof sender is not same as producerPubkey", bft.GroupId)
			return false, proofMap
		}

		//check if all req are the same as original req
		if !proto.Equal(proof.Req, bft.currProof.Req) {
			pcbft_log.Errorf("<%s> proof req is not same as original req", bft.GroupId)
			return false, proofMap
		}

		//check if all resp sign against the same original req
		if !proto.Equal(proof.Req, proof.Resp.Req) {
			pcbft_log.Errorf("<%s> proof req is not same as proof resp req", bft.GroupId)
			return false, proofMap
		}

		//check if all resp sign is valid
		dumpResp := &quorumpb.ChangeConsensusResp{
			RespId:       proof.Resp.RespId,
			GroupId:      proof.Resp.GroupId,
			SenderPubkey: proof.Resp.SenderPubkey,
			Req:          proof.Resp.Req,
			MsgHash:      nil,
			SenderSign:   nil,
		}

		byts, err := proto.Marshal(dumpResp)
		if err != nil {
			pcbft_log.Errorf("<%s> marshal change consensus resp failed", bft.GroupId)
			return false, proofMap
		}

		hash := localcrypto.Hash(byts)
		if !bytes.Equal(hash, proof.Resp.MsgHash) {
			pcbft_log.Errorf("<%s> proof resp hash is not same as original hash", bft.GroupId)
			return false, proofMap
		}

		isValid, err := bft.cIface.VerifySign(hash, proof.Resp.SenderSign, proof.Resp.SenderPubkey)
		if err != nil {
			pcbft_log.Errorf("<%s> verify proof resp sign failed", bft.GroupId)
			return false, proofMap
		}

		if !isValid {
			pcbft_log.Errorf("<%s> proof resp sign is not valid", bft.GroupId)
			return false, proofMap
		}
	}

	return true, proofMap
}
