package chain

import (
	"context"
	"fmt"

	"github.com/dgraph-io/badger/v3"
	"github.com/dgraph-io/badger/v3/options"
	localcrypto "github.com/huo-ju/quorum/internal/pkg/crypto"
	"github.com/huo-ju/quorum/internal/pkg/p2p"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	logging "github.com/ipfs/go-log/v2"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"google.golang.org/protobuf/proto"
)

var chainctx_log = logging.Logger("chainctx")

type NodeStatus int8

const (
	NODE_ONLINE  = 0 //node connected with bootstramp and pubchannel
	NODE_OFFLINE = 1 //node disconnected with bootstram and pubchannel
)

type NodeCtx struct {
	node      *p2p.Node
	PeerId    peer.ID
	Keystore  localcrypto.Keystore
	PublicKey p2pcrypto.PubKey
	Groups    map[string]*Group
	Name      string
	Ctx       context.Context
	Version   string
	Status    NodeStatus
}

var nodeCtx *NodeCtx

type DbOption struct {
	LogFileSize    int64
	MemTableSize   int64
	LogMaxEntries  uint32
	BlockCacheSize int64
	Compression    options.CompressionType
}

type DbMgr struct {
	GroupInfoDb *badger.DB
	Db          *badger.DB
	Auth        *badger.DB
	DataPath    string
}

var dbMgr *DbMgr

//singlaton
func GetNodeCtx() *NodeCtx {
	return nodeCtx
}

//singlaton
func GetDbMgr() *DbMgr {
	return dbMgr
}

func InitCtx(ctx context.Context, name string, node *p2p.Node, dataPath string, channeltype string, gitcommit string) {
	nodeCtx = &NodeCtx{}
	nodeCtx.Name = name
	nodeCtx.node = node
	dbMgr = &DbMgr{}
	nodeCtx.Groups = make(map[string]*Group)
	dbopts := &DbOption{LogFileSize: 16 << 20, MemTableSize: 8 << 20, LogMaxEntries: 50000, BlockCacheSize: 32 << 20, Compression: options.Snappy}
	dbMgr.InitDb(dataPath, dbopts)
	nodeCtx.Status = NODE_OFFLINE
	nodeCtx.Ctx = ctx
	nodeCtx.Version = "1.0.0"
}

func Release() {
	//close all groups
	for groupId, group := range nodeCtx.Groups {
		fmt.Println("group:", groupId, " teardown")
		group.Teardown()
	}
	//close ctx db
	dbMgr.CloseDb()
}

func (nodeCtx *NodeCtx) PeersProtocol() *map[string][]string {
	return nodeCtx.node.PeersProtocol()
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

//load and group and start syncing
func (nodeCtx *NodeCtx) SyncAllGroup() error {
	chain_log.Infof("Start Sync all groups")

	//open all groups
	groupItemsBytes, err := dbMgr.GetGroupsBytes()

	if err != nil {
		return err
	}

	for _, b := range groupItemsBytes {
		var group *Group
		group = &Group{}

		var item *quorumpb.GroupItem
		item = &quorumpb.GroupItem{}

		proto.Unmarshal(b, item)
		group.Init(item)
		if err == nil {
			chain_log.Infof(fmt.Sprintf("Start sync group: %s", item.GroupId))
			go group.StartSync()
			nodeCtx.Groups[item.GroupId] = group
		} else {
			chain_log.Infof(fmt.Sprintf("can't init group: %s", item.GroupId))
			chain_log.Fatalf(err.Error())
		}
	}

	return nil
}

func (nodeCtx *NodeCtx) StopSyncAllGroup() error {
	return nil
}
