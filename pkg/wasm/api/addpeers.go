package api

import "github.com/rumsystem/quorum/pkg/chainapi/handlers"

func AddPeers(peers []string) (*handlers.AddPeerResult, error) {
	return handlers.AddPeers(peers)
}
