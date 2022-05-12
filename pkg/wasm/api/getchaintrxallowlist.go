//go:build js && wasm
// +build js,wasm

package api

import (
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	quorumContext "github.com/rumsystem/quorum/pkg/wasm/context"
)

type ChainSendTrxRuleListItemResult struct {
	Data []*handlers.ChainSendTrxRuleListItem `json:"data"`
}

func GetChainTrxAllowList(groupId string) (*ChainSendTrxRuleListItemResult, error) {
	wasmCtx := quorumContext.GetWASMContext()
	res, err := handlers.GetChainTrxAllowList(wasmCtx.GetChainStorage(), groupId)
	if err != nil {
		return nil, err
	}
	return &ChainSendTrxRuleListItemResult{res}, nil
}
