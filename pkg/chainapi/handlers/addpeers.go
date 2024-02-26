package handlers

import (
	"github.com/libp2p/go-libp2p/core/peer"
	maddr "github.com/multiformats/go-multiaddr"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
)

// AddPeerParam list of multiaddr
type AddPeerParam []string // Example: ["/ip4/94.23.17.189/tcp/62777/p2p/16Uiu2HAm5waftP3s4oE1EzGF2SyWeK726P5B8BSgFJqSiz6xScGz", "/ip4/132.145.109.63/tcp/10666/p2p/16Uiu2HAmTovb8kAJiYK8saskzz7cRQhb45NRK5AsbtdmYsLfD3RM"]

type AddPeerResult struct {
	SuccCount int               `json:"succ_count" example:"100"`
	ErrCount  int               `json:"err_count" example:"20"`
	Errs      map[string]string `json:"error"` // Example: {"/ip4/132.145.109.63/tcp/10666/p2p/16Uiu2HAmTovb8kAJiYK8saskzz7cRQhb45NRK5AsbtdmYsLfD3RM": "error info"}

}

func AddPeers(input AddPeerParam) (*AddPeerResult, error) {
	peerserr := make(map[string]string)

	peersaddrinfo := []peer.AddrInfo{}
	for _, addr := range input {
		ma, err := maddr.NewMultiaddr(addr)
		if err != nil {
			peerserr[addr] = err.Error()
			continue
		}
		addrinfo, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			peerserr[addr] = err.Error()
			continue
		}
		peersaddrinfo = append(peersaddrinfo, *addrinfo)
	}

	result := &AddPeerResult{SuccCount: 0, ErrCount: len(peerserr), Errs: peerserr}

	if len(peersaddrinfo) > 0 {
		count := nodectx.GetNodeCtx().AddPeers(peersaddrinfo)
		result.SuccCount = count
	}
	return result, nil
}
