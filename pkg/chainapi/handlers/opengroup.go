package handlers

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/internal/pkg/appdata"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
)

type OpenGroupParam struct {
	GroupId string `from:"group_id" json:"group_id" validate:"required,uuid4" example:"ac0eea7c-2f3c-4c67-80b3-136e46b924a8"`
}

type OpenGroupResult struct {
	GroupId string `json:"group_id" validate:"required,uuid4" example:"ac0eea7c-2f3c-4c67-80b3-136e46b924a8"`
}

func OpenGroup(params *OpenGroupParam, appdb *appdata.AppDb) (*OpenGroupResult, error) {
	validate := validator.New()
	if err := validate.Struct(params); err != nil {
		return nil, err
	}

	groupmgr := chain.GetGroupMgr()
	_, ok := groupmgr.Groups[params.GroupId]

	if ok {
		return nil, fmt.Errorf("group <%s> already open", params.GroupId)
	}

	group := &chain.Group{}
	group.LoadGroupById(params.GroupId)

	//add to group mgr
	groupmgr.Groups[params.GroupId] = group
	return &OpenGroupResult{GroupId: params.GroupId}, nil
}
