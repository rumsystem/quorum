package api

import (
	"encoding/json"

	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

func Announce(data []byte) (*handlers.AnnounceResult, error) {
	params := &handlers.AnnounceParam{}
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, err
	}

	return handlers.AnnounceHandler(params)
}
