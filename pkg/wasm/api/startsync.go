package api

import (
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

func StartSync(groupId string) (*handlers.StartSyncResult, error) {
	return handlers.StartSync(groupId)
}
