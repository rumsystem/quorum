package relayapi

import "github.com/rumsystem/quorum/internal/pkg/storage"

type RelayServerHandler struct {
	db storage.QuorumStorage
}
