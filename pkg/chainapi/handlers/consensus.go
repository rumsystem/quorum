package handlers

import (
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type GetConsensusHistory struct {
	ConsensusHistory []*quorumpb.ChangeConsensusResultBundle `json:"consensus_history"`
}

func GetConsensusHistoryHandler(chainapidb def.APIHandlerIface, groupId string) (*GetConsensusHistory, error) {
	//get consensusbundle from db
	if groupId == "" {
		return nil, fmt.Errorf("group_id can't be nil.")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupId]; ok {
		consensusHistory, err := group.GetAllChangeConsensusResultBundle()
		if err != nil {
			return nil, err
		}
		resunlt := &GetConsensusHistory{
			ConsensusHistory: consensusHistory,
		}
		return resunlt, nil
	} else {
		return nil, fmt.Errorf("group <%s> not exist", groupId)
	}
}

type GetLatestConsensusChangeResult struct {
	LatestConsensusResult *quorumpb.ChangeConsensusResultBundle `json:"latest_consensus_result"`
}

func GetLatestConsensusChangeResultHandler(chainapidb def.APIHandlerIface, groupId string) (*GetLatestConsensusChangeResult, error) {
	return nil, nil
}

type GetConsensusResultByReqIdParam struct {
	ReqId string `json:"req_id" validate:"required"`
}

type GetConsensusResultByReqIdResult struct {
	ConsensusResult *quorumpb.ChangeConsensusResultBundle `json:"consensus_result"`
}

func GetConsensusResultByReqIdHandler(chainapidb def.APIHandlerIface, groupId string, reqId string) (*GetConsensusResultByReqIdResult, error) {
	return nil, nil
}

type GetCurrentConsensusParam struct {
}

type GetCurrentConsensusResult struct {
	Producers        []string                              `json:"producers"`
	TrxEpochInterval uint64                                `json:"trx_epoch_interval"`
	LatestConsensus  *quorumpb.ChangeConsensusResultBundle `json:"latest_consensus"`
}

func GetCurrentConsensusHandler(chainapidb def.APIHandlerIface, groupId string) (*GetCurrentConsensusResult, error) {
	return nil, nil
}
