package handlers

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/internal/pkg/appdata"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
)

type LeaveGroupParam struct {
	GroupId string `from:"group_id" json:"group_id" validate:"required"`
}

type LeaveGroupResult struct {
	GroupId string `json:"group_id" validate:"required"`
}

func LeaveGroup(params *LeaveGroupParam, appdb *appdata.AppDb) (*LeaveGroupResult, error) {
	validate := validator.New()
	if err := validate.Struct(params); err != nil {
		return nil, err
	}

	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[params.GroupId]
	if !ok {
		return nil, fmt.Errorf("Group %s not exist", params.GroupId)
	}

	if err := group.LeaveGrp(); err != nil {
		return nil, err
	}

	if err := group.ClearGroupData(); err != nil {
		return nil, err
	}

	// delete group seed from appdata
	if err := appdb.DelGroupSeed(params.GroupId); err != nil {
		return nil, fmt.Errorf("save group seed failed: %s", err)
	}

	delete(groupmgr.Groups, params.GroupId)

	return &LeaveGroupResult{GroupId: params.GroupId}, nil
}
