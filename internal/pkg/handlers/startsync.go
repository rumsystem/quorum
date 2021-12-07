package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chain"
)

type StartSyncResult struct {
	GroupId string `validate:"required"`
	Error   string
}

func StartSync(groupid string) (*StartSyncResult, error) {
	if groupid == "" {
		return nil, errors.New("group_id can't be nil.")
	}

	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[groupid]
	if !ok {
		return nil, fmt.Errorf("Group %s not exist", groupid)
	}

	if group.ChainCtx.Syncer.Status == chain.SYNCING_BACKWARD || group.ChainCtx.Syncer.Status == chain.SYNCING_FORWARD {
		errorInfo := "GROUP_ALREADY_IN_SYNCING"
		return nil, fmt.Errorf(errorInfo)
	}

	startSyncResult := &StartSyncResult{GroupId: group.Item.GroupId, Error: ""}
	if err := group.StartSync(); err != nil {
		startSyncResult.Error = err.Error()
	}
	return startSyncResult, nil
}
