package handlers

import (
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
)

type StartSyncResult struct {
	GroupId string `validate:"required"`
	Error   string
}

func StartSync(groupid string) (*StartSyncResult, error) {
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
}
