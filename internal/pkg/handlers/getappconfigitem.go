package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chain"
)

type AppConfigKeyItem struct {
	Name        string
	Type        string
	Value       string
	OwnerPubkey string
	OwnerSign   string
	Memo        string
	TimeStamp   int64
}

func GetAppConfigKey(itemKey, groupId string) (*AppConfigKeyItem, error) {
	if groupId == "" {
		return nil, errors.New("group_id can't be nil.")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupId]; ok {
		configItem, err := group.GetAppConfigItem(itemKey)
		if err != nil {
			return nil, err
		}
		var item *AppConfigKeyItem
		item = &AppConfigKeyItem{}

		item.Name = configItem.Name
		item.Type = configItem.Type.String()
		item.Value = configItem.Value
		item.OwnerPubkey = configItem.OwnerPubkey
		item.OwnerSign = configItem.OwnerSign
		item.Memo = configItem.Memo
		item.TimeStamp = configItem.TimeStamp
		return item, nil
	} else {
		return nil, fmt.Errorf("Group %s not exist", groupId)
	}
}
