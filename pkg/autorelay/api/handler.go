package api

import (
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p"
	"github.com/rumsystem/quorum/internal/pkg/storage"
)

type RelayServerHandler struct {
	db   storage.QuorumStorage
	node *p2p.RelayNode
}

func NewRelayServerHandler(db storage.QuorumStorage, node *p2p.RelayNode) RelayServerHandler {
	return RelayServerHandler{db, node}
}
