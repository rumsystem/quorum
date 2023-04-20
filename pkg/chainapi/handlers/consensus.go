package handlers

import (
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type GetConsensusHistory struct {
	Proofs []*quorumpb.ChangeConsensusResultBundle `json:"proofs"`
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
		resunlt := &GetConsensusHistory{
			Proofs: consensusHistory,
		}
		return resunlt, nil
	} else {
		return nil, fmt.Errorf("group <%s> not exist", groupId)
	}
}

type GetLatestConsensusChangeResult struct {
	Proof *quorumpb.ChangeConsensusResultBundle `json:"proof"`
}

func GetLatestConsensusChangeResultHandler(chainapidb def.APIHandlerIface, groupId string) (*GetLatestConsensusChangeResult, error) {
	if groupId == "" {
		return nil, fmt.Errorf("group_id can't be nil")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupId]; ok {
		latestConsensusResult, err := group.GetLastChangeConsensusResult(false)
		if err != nil {
			return nil, err
		}
		resunlt := &GetLatestConsensusChangeResult{
			Proof: latestConsensusResult,
		}
		return resunlt, nil
	}
	return nil, fmt.Errorf("group <%s> not exist", groupId)
}

type GetConsensusResultByReqIdParam struct {
	ReqId string `json:"req_id" validate:"required"`
}

type GetConsensusResultByReqIdResult struct {
	Proof *quorumpb.ChangeConsensusResultBundle `json:"proof"`
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
			Proof: consensusResult,
		}
		return resunlt, nil
	}
	return nil, fmt.Errorf("group <%s> not exist", groupId)
}

type GetCurrentConsensusResult struct {
	Producers        []*quorumpb.ProducerItem              `json:"producers"`
	TrxEpochInterval uint64                                `json:"trx_epoch_interval"`
	Proof            *quorumpb.ChangeConsensusResultBundle `json:"proof"`
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
		return &GetCurrentConsensusResult{Producers: producers, TrxEpochInterval: trxEpochInterval, Proof: proof}, nil
	}

	return nil, fmt.Errorf("group <%s> not exist", groupId)
}
