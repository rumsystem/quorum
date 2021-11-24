package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chain"
)

type DeniedUserListItem struct {
	GroupId          string `validate:"required"`
	PeerId           string `validate:"required"`
	GroupOwnerPubkey string `validate:"required"`
	GroupOwnerSign   string `validate:"required"`
	TimeStamp        int64  `validate:"required"`
	Action           string `validate:"required"`
	Memo             string `validate:"required"`
}

func GetDeniedUserList(groupid string) ([]*DeniedUserListItem, error) {
	if groupid == "" {
		return nil, errors.New("group_id can't be nil.")
	}
	var result []*DeniedUserListItem

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		blkList, err := group.GetBlockedUser()
		println("len: ", len(blkList))
		if err != nil {
			return nil, err
		}
		for _, blkItem := range blkList {
			var item *DeniedUserListItem
			item = &DeniedUserListItem{}

			item.GroupId = blkItem.GroupId
			item.PeerId = blkItem.PeerId
			item.GroupOwnerPubkey = blkItem.GroupOwnerPubkey
			item.GroupOwnerSign = blkItem.GroupOwnerSign
			item.Action = blkItem.Action
			item.Memo = blkItem.Memo
			item.TimeStamp = blkItem.TimeStamp
			println(item.PeerId)
			result = append(result, item)
		}
		return result, nil
	} else {
		return nil, fmt.Errorf("Group %s not exist", groupid)
	}
}
