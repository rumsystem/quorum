package api

import (
	"encoding/json"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	quorumContext "github.com/rumsystem/quorum/pkg/wasm/context"
)

func GroupProducer(data []byte) (*handlers.GrpProducerResult, error) {
	wasmCtx := quorumContext.GetWASMContext()
	params := &handlers.GrpProducerParam{}
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, err
	}

	return handlers.GroupProducer(wasmCtx.GetChainStorage(), params)
}
