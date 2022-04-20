package api

import (
	"github.com/rumsystem/quorum/internal/pkg/chainsdk/handlers"
)

func StartSync(groupId string) (*handlers.StartSyncResult, error) {
	return handlers.StartSync(groupId)
}
