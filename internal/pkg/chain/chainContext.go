package chain

import (
	"encoding/json"
	//"fmt"

	"context"
	"github.com/dgraph-io/badger/v3"
	"github.com/golang/glog"
	"github.com/libp2p/go-libp2p-core/peer"

	blockstore "github.com/huo-ju/go-ipfs-blockstore"
	"github.com/huo-ju/quorum/internal/pkg/cli"
	//"github.com/huo-ju/quorum/internal/pkg/data"
	"github.com/huo-ju/quorum/internal/pkg/p2p"
	ds_sync "github.com/ipfs/go-datastore/sync"
	ds_badger "github.com/ipfs/go-ds-badger"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type ChainContext struct {
	PeerId     peer.ID
	Privatekey p2pcrypto.PrivKey
	PublicKey  p2pcrypto.PubKey

	Group map[string]GroupItem

	GroupInfoDb  *badger.DB
	TrxDb        *badger.DB
	ConfigDb     *badger.DB //maybe use global config DB???
	BlockStorage blockstore.Blockstore

	PublicTopic     *pubsub.Topic
	PublicSubscribe *pubsub.Subscription

	Ctx context.Context

	//test only, Trx should be saved to TrxDb
	TrxItem map[string]Trx
}

var chainContext *ChainContext

func InitContext() {
	chainContext = &ChainContext{}

	var group map[string]GroupItem
	group = make(map[string]GroupItem)
	chainContext.Group = group

	var trxItem map[string]Trx
	trxItem = make(map[string]Trx)
	chainContext.TrxItem = trxItem
}

//singlaton
func GetContext() *ChainContext {
	return chainContext
}

func (chainContext *ChainContext) InitDB(datapath string) {
	groupInfo, err := badger.Open(badger.DefaultOptions(datapath + "_groups"))
	if err != nil {
		glog.Fatalf(err.Error())
	}

	defer groupInfo.Close()

	config, err := badger.Open(badger.DefaultOptions(datapath + "_config"))
	if err != nil {
		glog.Fatal(err.Error())
	}
	defer config.Close()

	trx, err := badger.Open(badger.DefaultOptions(datapath + "_trx"))
	if err != nil {
		glog.Fatal(err.Error())
	}
	defer trx.Close()

	badgerstorage, err := ds_badger.NewDatastore(datapath+"_blocks", nil)
	bs := blockstore.NewBlockstore(ds_sync.MutexWrap(badgerstorage), "block")

	if err != nil {
		glog.Fatal((err).Error())
	}

	chainContext.GroupInfoDb = groupInfo
	chainContext.ConfigDb = config
	chainContext.TrxDb = trx
	chainContext.BlockStorage = bs
}

func (chainctx *ChainContext) JoinPublicChannel(node *p2p.Node, publicChannel string, ctx context.Context, config cli.Config) error {

	publicTopic, err := node.Pubsub.Join(publicChannel)

	if err != nil {
		glog.Fatalf("Join <%s> failed", publicChannel)
		glog.Fatalf(err.Error())
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

	go handlePublicChannel(ctx, config)
	return nil
}

func handlePublicChannel(ctx context.Context, config cli.Config) error {
	for {
		msg, err := chainContext.PublicSubscribe.Next(ctx)
		if err != nil {
			glog.Fatalf(err.Error())
			return err
		}

		var trxMsg TrxMsg
		err = json.Unmarshal(msg.Data, &trxMsg)

		if err != nil {
			glog.Fatalf(err.Error())
			return err
		}

		handleTrxMsg(trxMsg)
	}
}
