package api

import "github.com/rumsystem/quorum/internal/pkg/handlers"

func AddPeers(peers []string) (*handlers.AddPeerResult, error) {
	return handlers.AddPeers(peers)
}
