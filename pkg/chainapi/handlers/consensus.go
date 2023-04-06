package handlers

import (
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type GetConsensusHistoryParam struct {
}

type GetConsensusHistory struct {
	ConsensusHistory []*quorumpb.ChangeConsensusResultBundle `json:"consensus_history"`
}

func GetConsensusHistoryHandler(params *GetConsensusHistoryParam) (*GetConsensusHistory, error) {
	return nil, nil
}

type GetLatestConsensusChangeResultParam struct {
}

type GetLatestConsensusChangeResult struct {
	LatestConsensusResult *quorumpb.ChangeConsensusResultBundle `json:"latest_consensus_result"`
}

func GetLatestConsensusChangeResultHandler(params *GetLatestConsensusChangeResultParam) (*GetLatestConsensusChangeResult, error) {
	return nil, nil
}

type GetConsensusResultByReqIdParam struct {
	ReqId string `json:"req_id" validate:"required"`
}

type GetConsensusResultByReqIdResult struct {
	ConsensusResult *quorumpb.ChangeConsensusResultBundle `json:"consensus_result"`
}

func GetConsensusResultByReqIdHandler(params *GetConsensusResultByReqIdParam) (*GetConsensusResultByReqIdResult, error) {
	return nil, nil
}

type GetCurrentConsensusParam struct {
}

type GetCurrentConsensusResult struct {
	Producers        []string                              `json:"producers"`
	TrxEpochInterval uint64                                `json:"trx_epoch_interval"`
	LatestConsensus  *quorumpb.ChangeConsensusResultBundle `json:"latest_consensus"`
}

func GetCurrentConsensusHandler(params *GetCurrentConsensusParam) (*GetCurrentConsensusResult, error) {
	return nil, nil
}
