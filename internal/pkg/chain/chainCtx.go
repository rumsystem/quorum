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

	PublicTopic     *pubsub.Topic
	PublicSubscribe *pubsub.Subscription

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

func (chainctx *ChainCtx) JoinPublicChannel(node *p2p.Node, publicChannel string, ctx context.Context, config cli.Config) error {
	publicTopic, err := node.Pubsub.Join(publicChannel)

	if err != nil {
		glog.Infof("Join <%s> failed", publicChannel)
		return err
	} else {
		glog.Infof("Join <%s> done", publicChannel)
	}

	chainctx.PublicTopic = publicTopic

	sub, err := publicTopic.Subscribe()
	if err != nil {
		glog.Fatalf("Subscribe <%s> failed", publicChannel)
		glog.Fatalf(err.Error())
		return err
	} else {
		glog.Infof("Subscribe <%s> done", publicChannel)
	}

	chainctx.PublicSubscribe = sub
	chainctx.Ctx = ctx
	chainctx.Status = NODE_ONLINE

	go handlePublicChannel(ctx, config)

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

func handlePublicChannel(ctx context.Context, config cli.Config) error {

	for {
		msg, err := chainCtx.PublicSubscribe.Next(ctx)
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
					handleTrxMsg(trxMsg)
				} else {
					//glog.Info("Msg from myself")
				}
			}
		}
	}
}
