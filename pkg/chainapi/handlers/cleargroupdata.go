package handlers

import (
	"github.com/go-playground/validator/v10"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
)

type ClearGroupDataParam struct {
	GroupId string `from:"group_id" json:"group_id" validate:"required,uuid4" example:"ac0eea7c-2f3c-4c67-80b3-136e46b924a8"`
}

type ClearGroupDataResult struct {
	GroupId string `json:"group_id" validate:"required,uuid4" example:"ac0eea7c-2f3c-4c67-80b3-136e46b924a8"`
}

func ClearGroupData(params *ClearGroupDataParam) (*ClearGroupDataResult, error) {
	validate := validator.New()
	err := validate.Struct(params)
	if err != nil {
		return nil, err
	}

	groupmgr := chain.GetGroupMgr()
	isExist, err := groupmgr.IsGroupExist(chain.JOIN_BY_API, params.GroupId)
	if err != nil {
		return nil, rumerrors.NewBadRequestError(rumerrors.ErrClearJoinedGroup)
	}

	if !isExist {
		return nil, rumerrors.NewBadRequestError(rumerrors.ErrGroupNotFound)
	}

	nodename := nodectx.GetNodeCtx().Name
	err = nodectx.GetNodeCtx().GetChainStorage().RemoveGroupData(params.GroupId, nodename)
	return &ClearGroupDataResult{GroupId: params.GroupId}, err
}
