package api

import (
	"errors"

	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

func LeaveGroup(groupId string) (*handlers.LeaveGroupResult, error) {
	if groupId == "" {
		return nil, errors.New("empty group id")
	}
	params := handlers.LeaveGroupParam{GroupId: groupId}
	return handlers.LeaveGroup(&params)
}
