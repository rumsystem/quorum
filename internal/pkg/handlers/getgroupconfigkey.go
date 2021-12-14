package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chain"
)

type GroupConfigKeyListItem struct {
	Name string
	Type string
}

func GetGroupConfigKeyList(groupId string) ([]*GroupConfigKeyListItem, error) {
	if groupId == "" {
		return nil, errors.New("group_id can't be nil.")
	}

	var result []*GroupConfigKeyListItem

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupId]; ok {
		nameList, typeList, err := group.GetGroupConfigKeyList()
		if err != nil {
			return nil, err
		}
		for i := range nameList {
			var item *GroupConfigKeyListItem
			item = &GroupConfigKeyListItem{}

			item.Name = nameList[i]
			item.Type = typeList[i]
			result = append(result, item)
		}
		return result, nil
	} else {
		return nil, fmt.Errorf("Group %s not exist", groupId)
	}
}
