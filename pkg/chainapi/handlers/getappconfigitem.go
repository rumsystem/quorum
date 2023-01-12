package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
)

type AppConfigKeyItem struct {
	Name        string `example:"test_string"`
	Type        string `example:"STRING"`
	Value       string `example:"123"`
	OwnerPubkey string `example:"CAISIQJOfMIyaYuVpzdeXq5p+ku/8pSB6XEmUJfHIJ3A0wCkIg=="`
	OwnerSign   string `example:"304502210091dcc8d8e167c128ef59af1b6e2b2efece499043cc149014303b932485cde3240220427f81f2d7482df0d9a4ab2c019528b33776c73daf21ba98921ee6ff4417b1bc"`
	Memo        string `example:"memo"`
	TimeStamp   int64  `example:"1639518490895535600"`
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
