//go:build js && wasm
// +build js,wasm

package api

import (
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

func GetGroupConfigKey(itemKey, groupId string) (*handlers.GroupConfigKeyItem, error) {
	return handlers.GetGroupConfigKey(itemKey, groupId)
}
