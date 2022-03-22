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
	Memo                   string
	TimeStamp              int64
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
			item.Memo = usr.Memo
			item.TimeStamp = usr.TimeStamp
			usrResultList = append(usrResultList, item)
		}

		return usrResultList, nil
	} else {
		return nil, fmt.Errorf("Group %s not exist", groupid)
	}
}

func GetAnnouncedGroupUser(groupid string, pubkey string) (*AnnouncedUserListItem, error) {
	if groupid == "" || pubkey == "" {
		return nil, errors.New("group_id or sign_pubkey can't be nil.")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {

		usr, err := group.GetAnnouncedUser(pubkey)
		if err != nil {
			return nil, err
		}

		var item *AnnouncedUserListItem
		item = &AnnouncedUserListItem{}
		item.AnnouncedSignPubkey = usr.SignPubkey
		item.AnnouncedEncryptPubkey = usr.EncryptPubkey
		item.AnnouncerSign = usr.AnnouncerSignature
		item.Result = usr.Result.String()
		item.Memo = usr.Memo
		item.TimeStamp = usr.TimeStamp

		return item, nil
	} else {
		return nil, fmt.Errorf("Group %s not exist", groupid)
	}
}
