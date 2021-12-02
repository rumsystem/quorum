package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chain"
)

type AnnouncedProducerListItem struct {
	AnnouncedPubkey string `validate:"required"`
	AnnouncerSign   string `validate:"required"`
	Result          string `validate:"required"`
	Action          string `validate:"required"`
	Memo            string `validate:"required"`
	TimeStamp       int64  `validate:"required"`
}

func GetAnnouncedGroupProducer(groupid string) ([]*AnnouncedProducerListItem, error) {
	if groupid == "" {
		return nil, errors.New("group_id can't be nil.")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		prdList, err := group.GetAnnouncedProducers()
		if err != nil {
			return nil, err
		}

		prdResultList := []*AnnouncedProducerListItem{}
		for _, prd := range prdList {
			var item *AnnouncedProducerListItem
			item = &AnnouncedProducerListItem{}
			item.AnnouncedPubkey = prd.SignPubkey
			item.AnnouncerSign = prd.AnnouncerSignature
			item.Result = prd.Result.String()
			item.Action = prd.Action.String()
			item.TimeStamp = prd.TimeStamp
			item.Memo = prd.Memo
			prdResultList = append(prdResultList, item)
		}

		return prdResultList, nil
	} else {
		return nil, fmt.Errorf("Group %s not exist", groupid)
	}
}
