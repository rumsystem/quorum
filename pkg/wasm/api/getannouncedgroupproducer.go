package api

import "github.com/rumsystem/quorum/internal/pkg/handlers"

type AnnouncedGroupProducerList struct {
	Data []*handlers.AnnouncedProducerListItem `json:"data"`
}

func GetAnnouncedGroupProducers(groupId string) (*AnnouncedGroupProducerList, error) {
	res, err := handlers.GetAnnouncedGroupProducer(groupId)
	if err != nil {
		return nil, err
	}
	return &AnnouncedGroupProducerList{res}, nil
}
