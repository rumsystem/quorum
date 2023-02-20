package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
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
			item := &AppConfigKeyListItem{
				Name: nameList[i],
				Type: typeList[i],
			}

			result = append(result, item)
		}
		return result, nil
	} else {
		return nil, fmt.Errorf("Group %s not exist", groupId)
	}
}
