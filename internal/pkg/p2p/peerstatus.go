package p2p

import (
	"fmt"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

type StatusType uint

const (
	PROTOCOL_NOT_SUPPORT StatusType = iota
)

type PeerStatus struct {
	status map[string]StatusType
}

func NewPeerStatus() *PeerStatus {
	status := make(map[string]StatusType)
	return &PeerStatus{status}
}

func (peerstat *PeerStatus) IfSkip(peerid peer.ID, protocol protocol.ID) bool {
	key := fmt.Sprintf("%s-%s", peerid, protocol)
	r, ok := peerstat.status[key]
	if ok == true && r == PROTOCOL_NOT_SUPPORT {
		return true
	}
	return false
}

func (peerstat *PeerStatus) Update(peerid peer.ID, protocol protocol.ID, stat StatusType) {
	if peerstat.status != nil {
		key := fmt.Sprintf("%s-%s", peerid, protocol)
		peerstat.status[key] = stat
	}
}
