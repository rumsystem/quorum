package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
)

type AnnouncedUserListItem struct {
	AnnouncedSignPubkey    string `example:"CAISIQIWQX/5Nmy2/YoBbdO9jn4tDgn22prqOWMYusBR6axenw=="`
	AnnouncedEncryptPubkey string `example:"age1a68u5gafkt3yfsz7pr45j5ku3tyyk4xh9ydp3xwpaphksz54kgns99me0g"`
	AnnouncerSign          string `example:"30450221009974a5e0f3ea114de8469a806894410d12b5dc5d6d7ee21e49b5482cb062f1740220168185ad84777675ba29773942596f2db0fa5dd810185d2b8113ac0eaf4d7603"`
	Result                 string `example:"ANNOUNCED"`
	Memo                   string `example:"Memo"`
	TimeStamp              int64  `example:"1642609852758917000"`
}

func GetAnnouncedGroupUsers(chainapidb def.APIHandlerIface, groupid string) ([]*AnnouncedUserListItem, error) {
	if groupid == "" {
		return nil, errors.New("group_id can't be nil.")
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		usrList, err := chainapidb.GetAnnounceUsersByGroup(group.GroupId, group.Nodename)
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
