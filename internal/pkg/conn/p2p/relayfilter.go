package p2p

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/pkg/relayapi/handlers"

	ma "github.com/multiformats/go-multiaddr"
)

type QuorumRelayFilter struct {
	db storage.QuorumStorage
}

func NewQuorumRelayFilter(db storage.QuorumStorage) *QuorumRelayFilter {
	rf := QuorumRelayFilter{db}
	return &rf
}

func (rf *QuorumRelayFilter) AllowReserve(p peer.ID, a ma.Multiaddr) bool {
	// we always allow reservation
	return true
}

func (rf *QuorumRelayFilter) AllowConnect(src peer.ID, srcAddr ma.Multiaddr, dest peer.ID) bool {
	// once traffic of a dest peer exceeds its limit, we dont allow connect to the peer anymore
	permission, err := handlers.GetPermissions(rf.db, dest.String())
	if err != nil {
		networklog.Errorf("getDestConnectPermission failed: %s:", err.Error())
		return false
	}

	// TODO: we could also add limitation for src peers
	return permission.AllowConnect
}
