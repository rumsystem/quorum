package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
)

type PostToGroupParam struct {
	GroupId string `param:"group_id" json:"group_id" validate:"required,uuid4" example:"ac0eea7c-2f3c-4c67-80b3-136e46b924a8"`
	Data    []byte
}

type TrxResult struct {
	TrxId string `json:"trx_id" validate:"required,uuid4" example:"9e54c173-c1dd-429d-91fa-a6b43c14da77"`
}

func PostToGroup(payload *PostToGroupParam) (*TrxResult, error) {
	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[payload.GroupId]
	if !ok {
		return nil, errors.New(fmt.Sprintf("Group with group_id <%s> not exist", payload.GroupId))
	}

	trxId, err := group.PostToGroup(payload.Data)
	if err != nil {
		return nil, err
	}
	return &TrxResult{trxId}, nil
}
