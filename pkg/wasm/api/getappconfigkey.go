//go:build js && wasm
// +build js,wasm

package api

import (
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

type AppConfigKeyList struct {
	Data []*handlers.AppConfigKeyListItem `json:"data"`
}

func GetAppConfigKeyList(groupId string) (*AppConfigKeyList, error) {
	res, err := handlers.GetAppConfigKeyList(groupId)
	if err != nil {
		return nil, err
	}
	return &AppConfigKeyList{res}, nil
}
