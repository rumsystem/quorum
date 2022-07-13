package api

import "github.com/rumsystem/quorum/internal/pkg/storage"

type RelayServerHandler struct {
	db storage.QuorumStorage
}

func NewRelayServerHandler(db storage.QuorumStorage) RelayServerHandler {
	h := RelayServerHandler{}
	h.db = db
	return h
}
