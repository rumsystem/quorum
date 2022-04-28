package api

import (
	"encoding/json"

	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

func UpdateProfile(data []byte) (*handlers.UpdateProfileResult, error) {
	paramspb := new(quorumpb.Activity)
	if err := json.Unmarshal(data, &paramspb); err != nil {
		return nil, err
	}

	return handlers.UpdateProfile(paramspb)
}
