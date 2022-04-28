//go:build js && wasm
// +build js,wasm

package api

import "github.com/rumsystem/quorum/pkg/chainapi/handlers"

func GetChainTrxAuthMode(groupId, trxType string) (*handlers.TrxAuthItem, error) {
	return handlers.GetChainTrxAuthMode(groupId, trxType)
}
