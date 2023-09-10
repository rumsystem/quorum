package handlers

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
)

type CloseGroupParam struct {
	GroupId string `from:"group_id" json:"group_id" validate:"required,uuid4" example:"ac0eea7c-2f3c-4c67-80b3-136e46b924a8"`
}

type CloseGroupResult struct {
	GroupId string `json:"group_id" validate:"required,uuid4" example:"ac0eea7c-2f3c-4c67-80b3-136e46b924a8"`
}

// @Tags Groups
// @Summary CloseGroup
// @Description Close a group
// @Accept json
// @Produce json
// @Param data body handlers.LeaveGroupParam true "LeaveGroupParam"
// @success 200 {object} handlers.LeaveGroupResult "LeaveGroupResult"
// @Router /api/v2/group/close [post]
func CloseGroup(params *CloseGroupParam) (*CloseGroupResult, error) {
	validate := validator.New()
	if err := validate.Struct(params); err != nil {
		return nil, err
	}

	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[params.GroupId]
	if !ok {
		return nil, fmt.Errorf("group <%s> not exist", params.GroupId)
	}

	group.StopSync()
	if err := group.Teardown(); err != nil {
		return nil, err
	}

	delete(groupmgr.Groups, params.GroupId)
	return &CloseGroupResult{GroupId: params.GroupId}, nil
}
