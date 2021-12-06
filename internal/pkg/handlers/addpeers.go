package handlers

import (
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
	maddr "github.com/multiformats/go-multiaddr"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
)

type AddPeerParam []string

type AddPeerResult struct {
	SuccCount int `json:"succ_count"`
	ErrCount  int `json:"err_count"`
}

func AddPeers(input AddPeerParam) (*AddPeerResult, error) {
	peerserr := make(map[string]string)

	peersaddrinfo := []peer.AddrInfo{}
	for _, addr := range input {
		ma, err := maddr.NewMultiaddr(addr)
		if err != nil {
			peerserr[addr] = fmt.Sprintf("%s", err)
			continue
		}
		addrinfo, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			peerserr[addr] = fmt.Sprintf("%s", err)
			continue
		}
		peersaddrinfo = append(peersaddrinfo, *addrinfo)
	}

	result := &AddPeerResult{SuccCount: 0, ErrCount: len(peerserr)}

	if len(peersaddrinfo) > 0 {
		count := nodectx.GetNodeCtx().AddPeers(peersaddrinfo)
		result.SuccCount = count
	}
	return result, nil
}
