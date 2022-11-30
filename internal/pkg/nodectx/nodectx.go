package nodectx

import (
	"context"

	p2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	chainstorage "github.com/rumsystem/quorum/internal/pkg/storage/chain"
	"github.com/rumsystem/quorum/pkg/constants"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
)

type NodeStatus int8

const (
	NODE_ONLINE  = 0 //node connected with bootstramp and pubchannel
	NODE_OFFLINE = 1 //node disconnected with bootstram and pubchannel
)

type NODE_TYPE int

const (
	BOOTSTRAP_NODE NODE_TYPE = iota
	PRODUCER_NODE
	FULL_NODE
)

type NodeCtx struct {
	Node      *p2p.Node
	NodeType  NODE_TYPE
	PeerId    peer.ID
	Keystore  localcrypto.Keystore
	PublicKey p2pcrypto.PubKey
	Name      string
	Ctx       context.Context
	Version   string
	Status    NodeStatus
	chaindb   *chainstorage.Storage
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

func (nodeCtx *NodeCtx) GetChainStorage() *chainstorage.Storage {
	return nodeCtx.chaindb
}

func InitCtx(ctx context.Context, name string, node *p2p.Node, db *storage.DbMgr, chaindb *chainstorage.Storage, channeltype string, gitcommit string, nodetype NODE_TYPE) {
	nodeCtx = &NodeCtx{}
	nodeCtx.Name = name
	nodeCtx.Node = node
	nodeCtx.chaindb = chaindb
	nodeCtx.NodeType = nodetype

	dbMgr = db

	nodeCtx.Status = NODE_OFFLINE
	nodeCtx.Ctx = ctx
	nodeCtx.Version = "2.0.0"
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

func (nodeCtx *NodeCtx) ListGroupPeers(groupid string) []peer.ID {
	userChannelId := constants.USER_CHANNEL_PREFIX + groupid
	return nodeCtx.Node.Pubsub.ListPeers(userChannelId)
}

func (nodeCtx *NodeCtx) AddPeers(peers []peer.AddrInfo) int {
	return nodeCtx.Node.AddPeers(nodeCtx.Ctx, peers)
}
