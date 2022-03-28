//go:build js && wasm
// +build js,wasm

package api

import (
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

func GetPubQueue(groupId string) (*handlers.PubQueueInfo, error) {
	return handlers.GetPubQueue(groupId)
}
