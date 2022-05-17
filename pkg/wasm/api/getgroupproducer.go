package api

import (
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	quorumContext "github.com/rumsystem/quorum/pkg/wasm/context"
)

type ProducerList struct {
	Data []*handlers.ProducerListItem `json:"data"`
}

func GetGroupProducers(groupId string) (*ProducerList, error) {
	wasmCtx := quorumContext.GetWASMContext()
	res, err := handlers.GetGroupProducers(wasmCtx.GetChainStorage(), groupId)
	if err != nil {
		return nil, err
	}
	return &ProducerList{res}, nil
}
