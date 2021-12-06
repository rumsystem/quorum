package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chain"
)

type AnnouncedUserListItem struct {
	AnnouncedSignPubkey    string
	AnnouncedEncryptPubkey string
	AnnouncerSign          string
	Result                 string
}

func GetAnnouncedGroupUsers(groupid string) ([]*AnnouncedUserListItem, error) {
	if groupid == "" {
		return nil, errors.New("group_id can't be nil.")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		usrList, err := group.GetAnnouncedUsers()
		if err != nil {
			return nil, err
		}

		usrResultList := []*AnnouncedUserListItem{}
		for _, usr := range usrList {
			var item *AnnouncedUserListItem
			item = &AnnouncedUserListItem{}
			item.AnnouncedSignPubkey = usr.SignPubkey
			item.AnnouncedEncryptPubkey = usr.EncryptPubkey
			item.AnnouncerSign = usr.AnnouncerSignature
			item.Result = usr.Result.String()
			usrResultList = append(usrResultList, item)
		}

		return usrResultList, nil
	} else {
		return nil, fmt.Errorf("Group %s not exist", groupid)
	}
}
