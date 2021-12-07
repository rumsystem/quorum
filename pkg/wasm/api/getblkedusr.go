package api

import "github.com/rumsystem/quorum/internal/pkg/handlers"

type DeniedUserListResult struct {
	Data []*handlers.DeniedUserListItem `json:"data"`
}

func GetDeniedUserList(groupId string) (*DeniedUserListResult, error) {
	res, err := handlers.GetDeniedUserList(groupId)
	if err != nil {
		return nil, err
	}
	return &DeniedUserListResult{res}, nil
}
