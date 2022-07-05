package p2p

import (
	"github.com/libp2p/go-libp2p-core/peer"

	ma "github.com/multiformats/go-multiaddr"
)

type QuorumRelayFilter struct {
	// TODO: read from db, db is synced from somewhere?
}

func NewQuorumRelayFilter() *QuorumRelayFilter {
	rf := QuorumRelayFilter{}
	return &rf
}

func (rf *QuorumRelayFilter) AllowReserve(p peer.ID, a ma.Multiaddr) bool {
	return true
}

func (rf *QuorumRelayFilter) AllowConnect(src peer.ID, srcAddr ma.Multiaddr, dest peer.ID) bool {
	return true
}
