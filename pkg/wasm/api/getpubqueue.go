//go:build js && wasm
// +build js,wasm

package api

import (
	"github.com/rumsystem/quorum/internal/pkg/chainsdk/handlers"
)

func GetPubQueue(groupId string, status string, trxId string) (*handlers.PubQueueInfo, error) {
	return handlers.GetPubQueue(groupId, status, trxId)
}

func PubQueueAck(trxIds []string) ([]string, error) {
	return handlers.PubQueueAck(trxIds)
}
