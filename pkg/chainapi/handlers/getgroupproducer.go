package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
)

type ProducerListItem struct {
	ProducerPubkey string `example:"CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ=="`
	OwnerPubkey    string `example:"CAISIQNVGW0jrrKvo9/40lAyz/uICsyBbk465PmDKdWfcCM4JA=="`
	OwnerSign      string `example:"304402202cbca750600cd0aeb3a1076e4aa20e9d1110fe706a553df90d0cd69289628eed022042188b48fa75d0197d9f5ce03499d3b95ffcdfb0ace707cf3eda9f12473db0ea"`
	TimeStamp      int64  `example:"1634756661280204800"`
	BlockWithness  int64  `example:"0"`
}

func GetGroupProducers(chainapidb def.APIHandlerIface, groupid string) ([]*ProducerListItem, error) {
	if groupid == "" {
		return nil, errors.New("group_id can't be nil.")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		prdList, err := chainapidb.GetProducers(group.GroupId, group.Nodename)
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
			item.BlockWithness = prd.WithnessBlockNum
			prdResultList = append(prdResultList, item)
		}

		return prdResultList, nil
	} else {
		return nil, fmt.Errorf("Group %s not exist", groupid)
	}
}
