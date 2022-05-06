package handlers

import (
	"github.com/libp2p/go-libp2p-core/peer"
	maddr "github.com/multiformats/go-multiaddr"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
)

type AddRelayParam []string

func AddRelayServers(input AddRelayParam) (bool, error) {
	peers := []peer.AddrInfo{}
	for _, addr := range input {
		ma, err := maddr.NewMultiaddr(addr)
		if err != nil {
			return false, err
		}
		addrinfo, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			return false, err
		}
		peers = append(peers, *addrinfo)
	}

	nodectx.GetNodeCtx().AddPeers(peers)

	peerChan := p2p.GetRelayPeerChan()

	for _, peer := range peers {
		peerChan <- peer
	}

	return true, nil
}
