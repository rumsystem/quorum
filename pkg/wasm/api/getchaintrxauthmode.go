//go:build js && wasm
// +build js,wasm

package api

import (
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	quorumContext "github.com/rumsystem/quorum/pkg/wasm/context"
)

func GetChainTrxAuthMode(groupId, trxType string) (*handlers.TrxAuthItem, error) {
	wasmCtx := quorumContext.GetWASMContext()
	return handlers.GetChainTrxAuthMode(wasmCtx.GetChainStorage(), groupId, trxType)
}
