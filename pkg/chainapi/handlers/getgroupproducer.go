package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type ProducerListItem struct {
	Prodeucers []*quorumpb.ProducerItem `json:"producers"`
}

func GetGroupProducers(chainapidb def.APIHandlerIface, groupid string) (*ProducerListItem, error) {
	if groupid == "" {
		return nil, errors.New("group_id can't be nil.")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		prdList, err := chainapidb.GetProducers(group.GroupId, group.Nodename)
		if err != nil {
			return nil, err
		}

		result := &ProducerListItem{
			Prodeucers: prdList,
		}

		return result, nil
	} else {
		return nil, fmt.Errorf("Group %s not exist", groupid)
	}
}
