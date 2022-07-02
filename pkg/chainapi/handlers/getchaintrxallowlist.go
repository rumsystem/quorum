package handlers

import (
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

type ChainSendTrxRuleListItem struct {
	Pubkey           string   `validate:"required"`
	TrxType          []string `validate:"required"`
	GroupOwnerPubkey string   `validate:"required"`
	GroupOwnerSign   string   `validate:"required"`
	TimeStamp        int64    `validate:"required"`
	Memo             string
}

func GetChainTrxAllowList(chainapidb def.APIHandlerIface, groupid string) ([]*ChainSendTrxRuleListItem, error) {
	if groupid == "" {
		return nil, rumerrors.ErrInvalidGroupID
	}
	var result []*ChainSendTrxRuleListItem

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		chainConfigItemList, allowItemList, err := chainapidb.GetSendTrxAuthListByGroupId(group.Item.GroupId, quorumpb.AuthListType_ALLOW_LIST, group.ChainCtx.GetNodeName())

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
		return nil, rumerrors.ErrGroupNotFound
	}
}
