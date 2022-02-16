//go:build js && wasm
// +build js,wasm

package api

import (
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

type GroupConfigKeyList struct {
	Data []*handlers.GroupConfigKeyListItem `json:"data"`
}

func GetGroupConfigKeyList(groupId string) (*GroupConfigKeyList, error) {
	res, err := handlers.GetGroupConfigKeyList(groupId)
	if err != nil {
		return nil, err
	}
	return &GroupConfigKeyList{res}, nil
}
