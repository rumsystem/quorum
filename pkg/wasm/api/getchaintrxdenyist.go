//go:build js && wasm
// +build js,wasm

package api

import "github.com/rumsystem/quorum/internal/pkg/chainsdk/handlers"

func GetChainTrxDenyList(groupId string) (*ChainSendTrxRuleListItemResult, error) {
	res, err := handlers.GetChainTrxDenyList(groupId)
	if err != nil {
		return nil, err
	}
	return &ChainSendTrxRuleListItemResult{res}, nil
}
