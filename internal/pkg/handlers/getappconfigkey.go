package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chain"
)

type AppConfigKeyListItem struct {
	Name string
	Type string
}

func GetAppConfigKeyList(groupId string) ([]*AppConfigKeyListItem, error) {
	if groupId == "" {
		return nil, errors.New("group_id can't be nil.")
	}

	result := []*AppConfigKeyListItem{}
	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupId]; ok {
		nameList, typeList, err := group.GetAppConfigKeyList()
		if err != nil {
			return nil, err
		}
		for i := range nameList {
			var item *AppConfigKeyListItem
			item = &AppConfigKeyListItem{}

			item.Name = nameList[i]
			item.Type = typeList[i]
			result = append(result, item)
		}
		return result, nil
	} else {
		return nil, fmt.Errorf("Group %s not exist", groupId)
	}
}
