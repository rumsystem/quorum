package p2p

import (
	"fmt"
	"strconv"

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
	// we always allow reservation
	return true
}

func (rf *QuorumRelayFilter) AllowConnect(src peer.ID, srcAddr ma.Multiaddr, dest peer.ID) bool {
	destPermission := rf.getDestConnectPermission(dest)

	// TODO: we could also add limitation for src peers
	return destPermission
}

func (rf *QuorumRelayFilter) getDestConnectPermission(dest peer.ID) bool {
	// once traffic of a dest peer exceeds its limit, we dont allow connect to the peer anymore
	k := []byte(fmt.Sprintf("AllowConnectTo_%s", dest.String()))

	isExist, err := rf.db.IsExist(k)
	if err != nil {
		networklog.Errorf("getDestConnectPermission failed: %s:", err.Error())
		return false
	}
	if !isExist {
		// allow by default
		return true
	}

	v, err := rf.db.Get(k)
	if err != nil {
		networklog.Errorf("getDestConnectPermission failed: %s:", err.Error())
		return false
	}
	ok, _ := strconv.ParseBool(string(v))

	return ok
}
