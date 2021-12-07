package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chain"
)

type ProducerListItem struct {
	ProducerPubkey string
	OwnerPubkey    string
	OwnerSign      string
	TimeStamp      int64
	BlockProduced  int64
}

func GetGroupProducers(groupid string) ([]*ProducerListItem, error) {
	if groupid == "" {
		return nil, errors.New("group_id can't be nil.")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		prdList, err := group.GetProducers()
		if err != nil {
			return nil, err
		}

		var prdResultList []*ProducerListItem
		for _, prd := range prdList {
			var item *ProducerListItem
			item = &ProducerListItem{}
			item.ProducerPubkey = prd.ProducerPubkey
			item.OwnerPubkey = prd.GroupOwnerPubkey
			item.OwnerSign = prd.GroupOwnerSign
			item.TimeStamp = prd.TimeStamp
			item.BlockProduced = prd.BlockProduced
			prdResultList = append(prdResultList, item)
		}

		return prdResultList, nil
	} else {
		return nil, fmt.Errorf("Group %s not exist", groupid)
	}
}
