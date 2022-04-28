package api

import (
	"encoding/json"

	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

func GroupProducer(data []byte) (*handlers.GrpProducerResult, error) {
	params := &handlers.GrpProducerParam{}
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, err
	}

	return handlers.GroupProducer(params)
}
