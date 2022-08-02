package api

import (
	relayv2 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/rumsystem/quorum/internal/pkg/storage"
)

type RelayServerHandler struct {
	db    storage.QuorumStorage
	Relay *relayv2.Relay
}

func NewRelayServerHandler(db storage.QuorumStorage, relay *relayv2.Relay) RelayServerHandler {
	return RelayServerHandler{db, relay}
}
