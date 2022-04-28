//go:build js && wasm
// +build js,wasm

package api

import (
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

func ClearGroupData(groupId string) (*handlers.ClearGroupDataResult, error) {
	params := &handlers.ClearGroupDataParam{GroupId: groupId}

	return handlers.ClearGroupData(params)
}
