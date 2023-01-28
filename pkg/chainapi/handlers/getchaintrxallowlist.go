package handlers

import (
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type ChainSendTrxRuleListItem struct {
	Pubkey           string   `validate:"required" example:"CAISIQNGAO67UTFSuWzySHKdy4IjBI/Q5XDMELPUSxHpBwQDcQ=="`
	TrxType          []string `validate:"required" example:"POST,ANNOUNCE,REQ_BLOCK_FORWARD,REQ_BLOCK_BACKWARD,ASK_PEERID"`
	GroupOwnerPubkey string   `validate:"required" example:"CAISIQPLW/J9xgdMWoJxFttChoGOOld8TpChnGFFyPADGL+0JA=="`
	GroupOwnerSign   string   `validate:"required" example:"304502210084bc833278dc98be6f279540b571ad5402f5c2d1e978c4c2298cddb079ca312002205f9374b9d27c628815aecff4ffe11329b17b8be12687223a072afa58e9f15f2c"`
	TimeStamp        int64    `validate:"required" example:"1642609852758917000"`
	Memo             string   `example:"Memo"`
}

func GetChainTrxAllowList(chainapidb def.APIHandlerIface, groupid string) ([]*ChainSendTrxRuleListItem, error) {
	if groupid == "" {
		return nil, rumerrors.ErrInvalidGroupID
	}
	var result []*ChainSendTrxRuleListItem

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		chainConfigItemList, allowItemList, err := chainapidb.GetSendTrxAuthListByGroupId(group.GroupId, quorumpb.AuthListType_ALLOW_LIST, group.Nodename)

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
