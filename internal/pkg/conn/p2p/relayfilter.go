package p2p

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/pkg/autorelay/handlers"

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
	// check wheter remote peer can connect to local peer
	// once traffic of a dest peer exceeds its limit, we dont allow connect to the peer anymore
	// ps: src peer is the remote peer, so dest peer is the server peer
	permission, err := handlers.GetPermissions(rf.db, dest.String())
	if err != nil {
		networklog.Errorf("getDestConnectPermission failed: %s:", err.Error())
		return false
	}

	if !permission.AllowConnect {
		// maybe server peer is out of money/traffic
		return false
	}

	// check whether the remote peer is in the blacklist of server peer
	// should check both side, cause connection could be bio connection
	inBlacklist, err := handlers.CheckBlacklist(rf.db, dest.String(), src.String())
	if err != nil {
		// db error, we abort connect by now
		return false
	}
	inBlacklistRev, err := handlers.CheckBlacklist(rf.db, src.String(), dest.String())
	if err != nil {
		// db error, we abort connect by now
		return false
	}

	return !inBlacklist && !inBlacklistRev
}
