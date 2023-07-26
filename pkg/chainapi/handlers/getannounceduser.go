package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"

	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type AnnouncedUsers struct {
	Users []*quorumpb.AnnounceItem `json:"users"`
}

func GetAnnouncedUsers(chainapidb def.APIHandlerIface, groupid string) (*AnnouncedUsers, error) {
	if groupid == "" {
		return nil, errors.New("group_id can't be nil")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		usrList, err := chainapidb.GetAnnouncedUsers(group.GroupId, group.Nodename)
		if err != nil {
			return nil, err
		}

		result := &AnnouncedUsers{
			Users: usrList,
		}
		return result, nil
	} else {
		return nil, fmt.Errorf("group <%s> not exist", groupid)
	}
}

func GetAnnouncedUser(chainapidb def.APIHandlerIface, groupid string, pubkey string) (*AnnouncedUsers, error) {
	if groupid == "" || pubkey == "" {
		return nil, errors.New("group_id or sign_pubkey can't be nil")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		usr, err := group.GetAnnouncedUser(pubkey)
		if err != nil {
			return nil, err
		}

		result := &AnnouncedUsers{
			Users: []*quorumpb.AnnounceItem{usr},
		}
		return result, nil
	} else {
		return nil, fmt.Errorf("group <%s> not exist", groupid)
	}
}
