package nodectx

import (
	"context"

	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/p2p"
	"github.com/rumsystem/quorum/internal/pkg/storage"
)

type NodeStatus int8

const (
	NODE_ONLINE  = 0 //node connected with bootstramp and pubchannel
	NODE_OFFLINE = 1 //node disconnected with bootstram and pubchannel
)

const (
	USER_CHANNEL_PREFIX     = "user_channel_"
	PRODUCER_CHANNEL_PREFIX = "prod_channel_"
	SYNC_CHANNEL_PREFIX     = "sync_channel_"
)

type NodeCtx struct {
	Node      *p2p.Node
	PeerId    peer.ID
	Keystore  localcrypto.Keystore
	PublicKey p2pcrypto.PubKey
	Name      string
	Ctx       context.Context
	Version   string
	Status    NodeStatus
}

var nodeCtx *NodeCtx

var dbMgr *storage.DbMgr

//singlaton
func GetNodeCtx() *NodeCtx {
	return nodeCtx
}

//singlaton
func GetDbMgr() *storage.DbMgr {
	return dbMgr
}

func InitCtx(ctx context.Context, name string, node *p2p.Node, db *storage.DbMgr, channeltype string, gitcommit string) {
	nodeCtx = &NodeCtx{}
	nodeCtx.Name = name
	nodeCtx.Node = node

	dbMgr = db

	nodeCtx.Status = NODE_OFFLINE
	nodeCtx.Ctx = ctx
	nodeCtx.Version = "1.0.0"
}

func (nodeCtx *NodeCtx) PeersProtocol() *map[string][]string {
	return nodeCtx.Node.PeersProtocol()
}

func (nodeCtx *NodeCtx) ProtocolPrefix() string {
	return p2p.ProtocolPrefix
}

func (nodeCtx *NodeCtx) UpdateOnlineStatus(status NodeStatus) {
	nodeCtx.Status = status
}

func (nodeCtx *NodeCtx) GetNodePubKey() (string, error) {
	var pubkey string
	pubkeybytes, err := p2pcrypto.MarshalPublicKey(nodeCtx.PublicKey)
	if err != nil {
		return pubkey, err
	}

	pubkey = p2pcrypto.ConfigEncodeKey(pubkeybytes)
	return pubkey, nil
}

func (nodeCtx *NodeCtx) ListGroupPeers(groupid string) []peer.ID {
	userChannelId := USER_CHANNEL_PREFIX + groupid
	return nodeCtx.Node.Pubsub.ListPeers(userChannelId)
}

func (nodeCtx *NodeCtx) AddPeers(peers []peer.AddrInfo) int {
	return nodeCtx.Node.AddPeers(nodeCtx.Ctx, peers)
}
