package api

import (
	"encoding/json"

	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

func MgrGrpBlkList(data []byte) (*handlers.DenyUserResult, error) {
	params := &handlers.DenyListParam{}
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, err
	}

	return handlers.MgrGrpBlkList(params)
}
