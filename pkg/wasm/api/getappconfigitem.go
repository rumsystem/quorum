//go:build js && wasm
// +build js,wasm

package api

import (
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

func GetAppConfigItem(itemKey, groupId string) (*handlers.AppConfigKeyItem, error) {
	return handlers.GetAppConfigKey(itemKey, groupId)
}
