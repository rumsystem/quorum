//go:build js && wasm
// +build js,wasm

package api

import (
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	"github.com/rumsystem/quorum/pkg/pb"
)

func GetTrx(groupId string, trxId string) (*pb.Trx, []int64, error) {
	return handlers.GetTrx(groupId, trxId)
}
