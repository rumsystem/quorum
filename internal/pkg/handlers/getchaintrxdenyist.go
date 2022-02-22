package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chain"
)

func GetChainTrxDenyList(groupid string) ([]*ChainSendTrxRuleListItem, error) {
	if groupid == "" {
		return nil, errors.New("group_id can't be nil.")
	}
	var result []*ChainSendTrxRuleListItem

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		chainConfigItem, denyItemList, err := group.GetChainSendTrxDenyList()
		if err != nil {
			return nil, err
		}
		for i, blkItem := range denyItemList {
			var item *ChainSendTrxRuleListItem
			item = &ChainSendTrxRuleListItem{}

			item.Pubkey = blkItem.Pubkey
			item.GroupOwnerPubkey = chainConfigItem[i].OwnerPubkey
			item.GroupOwnerSign = chainConfigItem[i].OwnerSignature
			for _, trxType := range blkItem.Type {
				item.TrxType = append(item.TrxType, trxType.String())
			}
			item.TimeStamp = chainConfigItem[i].TimeStamp
			item.Memo = chainConfigItem[i].Memo
			result = append(result, item)
		}
		return result, nil
	} else {
		return nil, fmt.Errorf("Group %s not exist", groupid)
	}
}
