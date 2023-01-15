package handlers

import (
	"encoding/base64"
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
)

type PostToGroupParam struct {
	GroupId string `param:"group_id" json:"group_id" validate:"required" example:"ac0eea7c-2f3c-4c67-80b3-136e46b924a8"`
	Sudo    bool   `json:"sudo" example:"false"`
	/* Example:
	{
		"type": "Create",
		"object": {
			"type": "Note",
			"id": 1,
			"content": "hello world"
		}
	}
	*/
	Data string `json:"data" validate:"required"`
}

type TrxResult struct {
	TrxId string `json:"trx_id" validate:"required" example:"9e54c173-c1dd-429d-91fa-a6b43c14da77"`
}

func PostToGroup(payload *PostToGroupParam, sudo bool) (*TrxResult, error) {
	data, err := base64.StdEncoding.DecodeString(payload.Data)
	if err != nil {
		return nil, err
	}

	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[payload.GroupId]
	if !ok {
		return nil, errors.New(fmt.Sprintf("Group %s not exist", payload.GroupId))
	}

	if sudo && (group.Item.UserSignPubkey != group.Item.OwnerPubKey) {
		return nil, errors.New("Only group owner can run sudo")
	}

	trxId, err := group.PostToGroup(data, sudo)
	if err != nil {
		return nil, err
	}
	return &TrxResult{trxId}, nil
}
