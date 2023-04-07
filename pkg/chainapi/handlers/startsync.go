package handlers

import (
	"fmt"
)

type StartSyncResult struct {
	GroupId string `validate:"required" example:"5ed3f9fe-81e2-450d-9146-7a329aac2b62"`
	Error   string `example:""`
}

func StartSync(groupid string) (*StartSyncResult, error) {
	/*
		if groupid == "" {
			return nil, fmt.Errorf("group_id can't be nil")
		}

		groupmgr := chain.GetGroupMgr()
		group, ok := groupmgr.Groups[groupid]
		if !ok {
			return nil, fmt.Errorf("group %s not exist", groupid)
		}

		startSyncResult := &StartSyncResult{GroupId: group.Item.GroupId, Error: ""}
		if err := group.StartSync(true); err != nil {
			startSyncResult.Error = err.Error()
		}

		return startSyncResult, nil
	*/

	return nil, fmt.Errorf("Deprecated")
}
