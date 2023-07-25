package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"

	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type AnnouncedProducers struct {
	Producers []*quorumpb.AnnounceItem `json:"producers"`
}

func GetAnnouncedProducers(chainapidb def.APIHandlerIface, groupid string) (*AnnouncedProducers, error) {
	if groupid == "" {
		return nil, errors.New("group_id can't be nil.")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		prdList, err := chainapidb.GetAnnouncedProducers(group.GroupId, group.Nodename)
		if err != nil {
			return nil, err
		}

		result := &AnnouncedProducers{
			Producers: prdList,
		}

		return result, nil
	} else {
		return nil, fmt.Errorf("group <%s> not exist", groupid)
	}
}

func GetAnnouncedProducer(chainapidb def.APIHandlerIface, groupid string, pubkey string) (*AnnouncedProducers, error) {
	if groupid == "" || pubkey == "" {
		return nil, errors.New("group_id or sign_pubkey can't be nil.")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		prd, err := group.GetAnnouncedProducer(pubkey)
		if err != nil {
			return nil, err
		}

		result := &AnnouncedProducers{
			Producers: []*quorumpb.AnnounceItem{prd},
		}
		return result, nil
	} else {
		return nil, fmt.Errorf("group <%s> not exist", groupid)
	}
}
