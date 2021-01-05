package chain

import (
	"github.com/libp2p/go-libp2p-core/peer"
)

func (nodeCtx *NodeCtx) ListGroupPeers(groupid string) []peer.ID {
	userChannelId := USER_CHANNEL_PREFIX + groupid
	return nodeCtx.node.Pubsub.ListPeers(userChannelId)
}

func (nodeCtx *NodeCtx) AddPeers(peers []peer.AddrInfo) int {
	return nodeCtx.node.AddPeers(nodeCtx.Ctx, peers)
}
