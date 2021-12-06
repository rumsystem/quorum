package api

import "github.com/rumsystem/quorum/internal/pkg/handlers"

type AnnouncedGroupUserList struct {
	Data []*handlers.AnnouncedUserListItem `json:"data"`
}

func GetAnnouncedGroupUsers(groupId string) (*AnnouncedGroupUserList, error) {
	res, err := handlers.GetAnnouncedGroupUsers(groupId)
	if err != nil {
		return nil, err
	}
	return &AnnouncedGroupUserList{res}, nil
}
