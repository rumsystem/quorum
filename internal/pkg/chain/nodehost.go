package chain

import (
	"github.com/libp2p/go-libp2p-core/peer"
)

func (chainctx *ChainCtx) ListGroupPeers(groupid string) []peer.ID {
	return chainctx.node.Pubsub.ListPeers(groupid)
}

func (chainctx *ChainCtx) AddPeers(peers []peer.AddrInfo) int {
	return chainctx.node.AddPeers(chainctx.Ctx, peers)
}
