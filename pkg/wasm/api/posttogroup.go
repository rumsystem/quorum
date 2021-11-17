package api

import (
	"encoding/json"

	"github.com/rumsystem/quorum/internal/pkg/handlers"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

func PostToGroup(data []byte) (*handlers.TrxResult, error) {
	paramspb := new(quorumpb.Activity)
	if err := json.Unmarshal(data, &paramspb); err != nil {
		return nil, err
	}

	return handlers.PostToGroup(paramspb)
}
