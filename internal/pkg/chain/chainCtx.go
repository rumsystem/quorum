package chain

import (
	"fmt"

	"context"

	"github.com/dgraph-io/badger/v3"
	"github.com/dgraph-io/badger/v3/options"
	"github.com/golang/glog"
	"github.com/libp2p/go-libp2p-core/peer"
	"google.golang.org/protobuf/proto"

	"github.com/huo-ju/quorum/internal/pkg/cli"
	"github.com/huo-ju/quorum/internal/pkg/p2p"

	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type NodeStatus int8

const (
	NODE_ONLINE  = 0 //node connected with bootstramp and pubchannel
	NODE_OFFLINE = 1 //node disconnected with bootstram and pubchannel
)

type ChainCtx struct {
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

func InitCtx(dataPath string) {
	chainCtx = &ChainCtx{}
	dbMgr = &DbMgr{}
	chainCtx.Groups = make(map[string]*Group)

	dbopts := &DbOption{LogFileSize: 16 << 20, MemTableSize: 8 << 20, LogMaxEntries: 50000, BlockCacheSize: 32 << 20, Compression: options.Snappy}
	dbMgr.InitDb(dataPath, dbopts)

	chainCtx.TrxSignReq = 1
	chainCtx.Status = NODE_OFFLINE
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

func (chainctx *ChainCtx) JoinGroupChannel(node *p2p.Node, groupId string, ctx context.Context, config cli.Config) error {
	groupTopic, err := node.Pubsub.Join(groupId)

	if err != nil {
		glog.Infof("Join <%s> failed", groupId)
		return err
	} else {
		glog.Infof("Join <%s> done", groupId)
	}

	chainctx.GroupTopics = append(chainctx.GroupTopics, groupTopic)

	sub, err := groupTopic.Subscribe()
	if err != nil {
		glog.Fatalf("Subscribe <%s> failed", groupId)
		glog.Fatalf(err.Error())
		return err
	} else {
		glog.Infof("Subscribe <%s> done", groupId)
	}

	chainctx.GroupSubscriptions = append(chainctx.GroupSubscriptions, sub)
	chainctx.Ctx = ctx
	chainctx.Status = NODE_ONLINE

	go handleGroupChannel(ctx, groupId, config)

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

//quit public channel before teardown
func (chainctx *ChainCtx) QuitPublicChannel() error {
	return nil
}

//load and group and start syncing
func (chainctx *ChainCtx) SyncAllGroup() error {
	glog.Infof("Start Sync all groups")

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
		go group.StartSync()
		chainctx.Groups[item.GroupId] = group
	}

	return nil
}

func (chainctx *ChainCtx) StopSyncAllGroup() error {
	return nil
}

func handleGroupChannel(ctx context.Context, groupId string, config cli.Config) error {
	var groupchannel *pubsub.Subscription
	for _, sub := range chainCtx.GroupSubscriptions {
		if sub.Topic() == groupId {
			groupchannel = sub
			break
		}
	}
	if groupchannel != nil {
		for {
			msg, err := groupchannel.Next(ctx)
			if err != nil {
				glog.Fatalf(err.Error())
				return err
			} else {
				var trxMsg quorumpb.TrxMsg
				err = proto.Unmarshal(msg.Data, &trxMsg)
				if err != nil {
					glog.Infof(err.Error())
				} else {
					if trxMsg.Version != GetChainCtx().Version {
						//glog.Infof("Version mismatch")
					} else if trxMsg.Sender != GetChainCtx().PeerId.Pretty() {
						handleTrxMsg(&trxMsg)
					} else {
						//glog.Info("Msg from myself")
					}
				}
			}
		}
	}
	return fmt.Errorf("unknown group topic: %s", groupId)
}
