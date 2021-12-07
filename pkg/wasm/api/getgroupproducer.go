package api

import "github.com/rumsystem/quorum/internal/pkg/handlers"

type ProducerList struct {
	Data []*handlers.ProducerListItem `json:"data"`
}

func GetGroupProducers(groupId string) (*ProducerList, error) {
	res, err := handlers.GetGroupProducers(groupId)
	if err != nil {
		return nil, err
	}
	return &ProducerList{res}, nil
}
