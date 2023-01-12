package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
)

type AnnouncedProducerListItem struct {
	AnnouncedPubkey string `validate:"required" example:"CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ=="`
	AnnouncerSign   string `validate:"required" example:"3046022100a853ca31f6f6719be213231b6428cecf64de5b1042dd8af1e140499507c85c40022100abd6828478f56da213ec10d361be8709333ff44cd0fa037409af9c0b67e6d0f5"`
	Result          string `validate:"required" example:"ANNOUCNED"`
	// ACTION have 2 states: "ADD" means Producer is normal, "REMOVE" means Producer has announced to leave the group by itself
	Action    string `validate:"required" example:"ADD"`
	Memo      string `validate:"required" example:"Memo"`
	TimeStamp int64  `validate:"required" example:"1634756064250457600"`
}

func GetAnnouncedGroupProducer(chainapidb def.APIHandlerIface, groupid string) ([]*AnnouncedProducerListItem, error) {
	if groupid == "" {
		return nil, errors.New("group_id can't be nil.")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		prdList, err := chainapidb.GetAnnounceProducersByGroup(group.GroupId, group.Nodename)
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
