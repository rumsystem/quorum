package handlers

import (
	"github.com/go-playground/validator/v10"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
)

type ReqPSyncResult struct {
	GroupId string `json:"group_id" validate:"required,uuid4"`
	ReqId   string `json:"req_id" validate:"required"`
}

type ReqPSyncParam struct {
	GroupId string `from:"group_id"    json:"group_id"    validate:"required,uuid4"`
}

func ReqPSyncHandler(params *ReqPSyncParam) (*ReqPSyncResult, error) {
	validate := validator.New()
	if err := validate.Struct(params); err != nil {
		return nil, err
	}
	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[params.GroupId]; !ok {
		return nil, rumerrors.ErrGroupNotFound
	} else {

		reqId, err := group.TryGetChainConsensus()
		if err != nil {
			return nil, err
		}

		return &ReqPSyncResult{GroupId: params.GroupId, ReqId: reqId}, nil
	}
}
