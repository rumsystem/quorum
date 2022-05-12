package api

import (
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	quorumContext "github.com/rumsystem/quorum/pkg/wasm/context"
)

type AnnouncedGroupProducerList struct {
	Data []*handlers.AnnouncedProducerListItem `json:"data"`
}

func GetAnnouncedGroupProducers(groupId string) (*AnnouncedGroupProducerList, error) {
	wasmCtx := quorumContext.GetWASMContext()
	res, err := handlers.GetAnnouncedGroupProducer(wasmCtx.GetChainStorage(), groupId)
	if err != nil {
		return nil, err
	}
	return &AnnouncedGroupProducerList{res}, nil
}
