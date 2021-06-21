package chain

import (
	"fmt"

	"context"

	"github.com/dgraph-io/badger/v3"
	"github.com/dgraph-io/badger/v3/options"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"google.golang.org/protobuf/proto"

	"github.com/huo-ju/quorum/internal/pkg/p2p"

	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

var chainctx_log = logging.Logger("chainctx")

type NodeStatus int8

const (
	NODE_ONLINE  = 0 //node connected with bootstramp and pubchannel
	NODE_OFFLINE = 1 //node disconnected with bootstram and pubchannel
)

type ChainCtx struct {
	node       *p2p.Node
	PeerId     peer.ID
	Privatekey p2pcrypto.PrivKey
	PublicKey  p2pcrypto.PubKey
	Groups     map[string]*Group

	GroupTopics        []*pubsub.Topic
	GroupSubscriptions []*pubsub.Subscription

	Ctx        context.Context
	TrxSignReq int

	Version string
	Status  NodeStatus
}

var chainCtx *ChainCtx

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
func GetChainCtx() *ChainCtx {
	return chainCtx
}

//singlaton
func GetDbMgr() *DbMgr {
	return dbMgr
}

func InitCtx(ctx context.Context, node *p2p.Node, dataPath string) {
	chainCtx = &ChainCtx{}
	chainCtx.node = node
	dbMgr = &DbMgr{}
	chainCtx.Groups = make(map[string]*Group)

	dbopts := &DbOption{LogFileSize: 16 << 20, MemTableSize: 8 << 20, LogMaxEntries: 50000, BlockCacheSize: 32 << 20, Compression: options.Snappy}
	dbMgr.InitDb(dataPath, dbopts)

	chainCtx.TrxSignReq = 1
	chainCtx.Status = NODE_OFFLINE
	chainCtx.Ctx = ctx
	chainCtx.Version = "ver 0.01"
}

func Release() {
	//close all groups
	for groupId, group := range chainCtx.Groups {
		fmt.Println("group:", groupId, " teardown")
		group.Teardown()
	}
	//close ctx db
	dbMgr.CloseDb()
}

func (chainctx *ChainCtx) Peers() *[]string {
	connectedpeers := []string{}
	peerstore := chainctx.node.Host.Peerstore()
	peers := peerstore.Peers()
	for _, peerid := range peers {
		if chainctx.node.Host.Network().Connectedness(peerid) == network.Connected {
			if chainctx.node.Host.ID() != peerid {
				connectedpeers = append(connectedpeers, peerid.Pretty())
			}
		}
	}
	return &connectedpeers

}
func (chainctx *ChainCtx) JoinGroupChannel(groupId string, ctx context.Context) error {
	var err error
	groupTopic := chainctx.GroupTopic(groupId)
	if groupTopic == nil {
		groupTopic, err = chainctx.node.Pubsub.Join(groupId)
		if err != nil {
			chain_log.Infof("Join <%s> failed", groupId)
			return err
		} else {
			chain_log.Infof("Join <%s> done", groupId)
		}
	}

	chainctx.GroupTopics = append(chainctx.GroupTopics, groupTopic)

	sub, err := groupTopic.Subscribe()
	if err != nil {
		chain_log.Fatalf("Subscribe <%s> failed", groupId)
		chain_log.Fatalf(err.Error())
		return err
	} else {
		chain_log.Infof("Subscribe <%s> done", groupId)
	}

	chainctx.GroupSubscriptions = append(chainctx.GroupSubscriptions, sub)
	//TODO: fix ONLINE status
	chainctx.Status = NODE_ONLINE

	go handleGroupChannel(ctx, sub, groupId)

	return nil
}

func (chainctx *ChainCtx) GroupTopic(groupId string) *pubsub.Topic {
	for _, topic := range chainctx.GroupTopics {
		if topic.String() == groupId {
			return topic
		}
	}
	return nil
}

func (chainctx *ChainCtx) GroupSubscription(groupId string) *pubsub.Subscription {
	for _, sub := range chainctx.GroupSubscriptions {
		if sub.Topic() == groupId {
			return sub
		}
	}
	return nil
}

func (chainctx *ChainCtx) GroupTopicPublish(groupId string, data []byte) error {
	grouptopic := chainctx.GroupTopic(groupId)
	if grouptopic != nil {
		return grouptopic.Publish(chainctx.Ctx, data)
	} else {
		return fmt.Errorf("can't publish to a unknown group topic: %s", groupId)
	}
}

//quit public channel before teardown
func (chainctx *ChainCtx) QuitPublicChannel() error {
	return nil
}

//load and group and start syncing
func (chainctx *ChainCtx) SyncAllGroup() error {
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
		group.init(item)
		err = chainctx.JoinGroupChannel(item.GroupId, context.Background())
		if err == nil {
			go group.StartSync()
			chainctx.Groups[item.GroupId] = group
		} else {
			chain_log.Infof(fmt.Sprintf("can't join group channel: %s", item.GroupId))
			chain_log.Fatalf(err.Error())

		}
	}

	return nil
}

func (chainctx *ChainCtx) StopSyncAllGroup() error {
	return nil
}

func handleGroupChannel(ctx context.Context, groupchannel *pubsub.Subscription, groupId string) error {
	if groupchannel != nil {
		for {
			msg, err := groupchannel.Next(ctx)
			if err == nil {
				var pkg quorumpb.Package
				err = proto.Unmarshal(msg.Data, &pkg)
				if err == nil {
					if pkg.Type == quorumpb.PackageType_BLOCK {
						//is block
						var blk quorumpb.Block
						err := proto.Unmarshal(pkg.Data, &blk)
						if err == nil {
							HandleBlock(&blk)
						} else {
							chain_log.Warning(err.Error())
						}
					} else if pkg.Type == quorumpb.PackageType_TRX {
						var trx quorumpb.Trx
						err := proto.Unmarshal(pkg.Data, &trx)
						if err == nil {
							if trx.Version != GetChainCtx().Version {
								chain_log.Infof("Version mismatch")
							} else if trx.Sender != GetChainCtx().PeerId.Pretty() {
								HandleTrx(&trx)
							} else {
								//chain_log.Info("Trx from myself")
							}
						} else {
							chain_log.Warning(err.Error())
						}
					}
				} else {
					chain_log.Warningf(err.Error())
				}
			} else {
				chain_log.Fatalf(err.Error())
				return err
			}
		}
	}
	return fmt.Errorf("unknown group topic: %s", groupId)
}
