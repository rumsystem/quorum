package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chain"
)

type ChainSendTrxRuleListItem struct {
	Pubkey           string   `validate:"required"`
	TrxType          []string `validate:"required"`
	GroupOwnerPubkey string   `validate:"required"`
	GroupOwnerSign   string   `validate:"required"`
	TimeStamp        int64    `validate:"required"`
	Memo             string   `validate:"required"`
}

func GetChainTrxAllowList(groupid string) ([]*ChainSendTrxRuleListItem, error) {
	if groupid == "" {
		return nil, errors.New("group_id can't be nil.")
	}
	var result []*ChainSendTrxRuleListItem

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		chainConfigItemList, allowItemList, err := group.GetChainSendTrxAllowList()
		if err != nil {
			return nil, err
		}
		for i, alwItem := range allowItemList {
			var item *ChainSendTrxRuleListItem
			item = &ChainSendTrxRuleListItem{}

			item.Pubkey = alwItem.Pubkey
			item.GroupOwnerPubkey = chainConfigItemList[i].OwnerPubkey
			item.GroupOwnerSign = chainConfigItemList[i].OwnerSignature
			for _, trxType := range alwItem.Type {
				item.TrxType = append(item.TrxType, trxType.String())
			}
			item.TimeStamp = chainConfigItemList[i].TimeStamp
			item.Memo = chainConfigItemList[i].Memo
			result = append(result, item)
		}
		return result, nil
	} else {
		return nil, fmt.Errorf("Group %s not exist", groupid)
	}
}
