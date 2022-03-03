//go:build js && wasm
// +build js,wasm

package api

import "github.com/rumsystem/quorum/internal/pkg/handlers"

type ChainSendTrxRuleListItemResult struct {
	Data []*handlers.ChainSendTrxRuleListItem `json:"data"`
}

func GetChainTrxAllowList(groupId string) (*ChainSendTrxRuleListItemResult, error) {
	res, err := handlers.GetChainTrxAllowList(groupId)
	if err != nil {
		return nil, err
	}
	return &ChainSendTrxRuleListItemResult{res}, nil
}
