package handlers

import (
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type GetConsensusHistory struct {
	Proofs []*ConsensusChangeResultBundle `json:"proofs"`
}

func GetConsensusHistoryHandler(chainapidb def.APIHandlerIface, groupId string) (*GetConsensusHistory, error) {
	//get consensusbundle from db
	if groupId == "" {
		return nil, fmt.Errorf("group_id can't be nil")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupId]; ok {
		consensusHistory, err := group.GetAllChangeConsensusResultBundle()
		if err != nil {
			return nil, err
		}

		var proofs []*ConsensusChangeResultBundle

		for _, item := range consensusHistory {
			proof := &ConsensusChangeResultBundle{
				Result:              item.Result.String(),
				Req:                 item.Req,
				RepononsedProducers: item.ResponsedProducers,
			}
			proofs = append(proofs, proof)
		}

		result := &GetConsensusHistory{
			Proofs: proofs,
		}
		return result, nil
	} else {
		return nil, fmt.Errorf("group <%s> not exist", groupId)
	}
}

type ConsensusChangeResultBundle struct {
	Result              string                       `json:"result"`
	Req                 *quorumpb.ChangeConsensusReq `json:"req"`
	RepononsedProducers []string                     `json:"repononsed_producer"`
}

func GetLatestConsensusChangeResultHandler(chainapidb def.APIHandlerIface, groupId string) (*ConsensusChangeResultBundle, error) {
	if groupId == "" {
		return nil, fmt.Errorf("group_id can't be nil")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupId]; ok {
		latestConsensusResult, err := group.GetLastChangeConsensusResult(false)
		if err != nil {
			return nil, err
		}

		result := &ConsensusChangeResultBundle{
			Result:              latestConsensusResult.Result.String(),
			Req:                 latestConsensusResult.Req,
			RepononsedProducers: latestConsensusResult.ResponsedProducers,
		}

		return result, nil
	}
	return nil, fmt.Errorf("group <%s> not exist", groupId)
}

type GetConsensusResultByReqIdParam struct {
	ReqId string `json:"req_id" validate:"required"`
}

type GetConsensusResultByReqIdResult struct {
	Result             string                          `json:"result"`
	Req                *quorumpb.ChangeConsensusReq    `json:"req"`
	Resps              []*quorumpb.ChangeConsensusResp `json:"resps"`
	ResponsedProducers []string                        `json:"responsed_producer"`
}

func GetConsensusResultByReqIdHandler(chainapidb def.APIHandlerIface, groupId string, reqId string) (*GetConsensusResultByReqIdResult, error) {
	if groupId == "" {
		return nil, fmt.Errorf("group_id can't be nil")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupId]; ok {
		consensusResult, err := group.GetChangeConsensusResultById(reqId)
		if err != nil {
			return nil, err
		}

		resunlt := &GetConsensusResultByReqIdResult{
			Result:             consensusResult.Result.String(),
			Req:                consensusResult.Req,
			Resps:              consensusResult.Resps,
			ResponsedProducers: consensusResult.ResponsedProducers,
		}
		return resunlt, nil
	}
	return nil, fmt.Errorf("group <%s> not exist", groupId)
}

type GetCurrentConsensusResult struct {
	Producers        []*quorumpb.ProducerItem `json:"producers"`
	TrxEpochInterval uint64                   `json:"trx_epoch_interval"`
	ProofReqID       string                   `json:"proof_req_id"`
	CurrEpoch        uint64                   `json:"curr_epoch"`
	CurrBlockId      uint64                   `json:"curr_block_id"`
	LastUpdate       int64                    `json:"last_update"`
}

func GetCurrentConsensusHandler(chainapidb def.APIHandlerIface, groupId string) (*GetCurrentConsensusResult, error) {
	if groupId == "" {
		return nil, fmt.Errorf("group_id can't be nil")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupId]; ok {
		producers, err := group.GetProducers()
		if err != nil {
			return nil, err
		}

		trxEpochInterval, err := group.GetCurrentTrxProposeInterval()
		if err != nil {
			return nil, err
		}
		proof, err := group.GetLastChangeConsensusResult(true)
		if err != nil {
			return nil, err
		}

		currEpoch := uint64(0)
		if group.IsOwner() || group.IsProducer() {
			currEpoch = group.GetCurrentEpoch()
		}

		currentBlockId := group.GetCurrentBlockId()
		lastUpdate := group.GetLatestUpdate()

		return &GetCurrentConsensusResult{Producers: producers,
			TrxEpochInterval: trxEpochInterval,
			ProofReqID:       proof.Req.ReqId,
			CurrEpoch:        currEpoch,
			CurrBlockId:      currentBlockId,
			LastUpdate:       lastUpdate}, nil
	}

	return nil, fmt.Errorf("group <%s> not exist", groupId)
}
