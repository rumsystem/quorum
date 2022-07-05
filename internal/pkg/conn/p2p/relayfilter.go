package p2p

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rumsystem/quorum/internal/pkg/storage"

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
	return true
}

func (rf *QuorumRelayFilter) AllowConnect(src peer.ID, srcAddr ma.Multiaddr, dest peer.ID) bool {
	return true
}
