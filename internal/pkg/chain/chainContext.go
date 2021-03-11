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

	Groups map[string]*GroupItem

	GroupInfoDb *badger.DB
	TrxDb       *badger.DB
	BlockDb     *badger.DB

	BlockStorage blockstore.Blockstore

	PublicTopic     *pubsub.Topic
	PublicSubscribe *pubsub.Subscription

	Ctx context.Context

	DataPath string
	Version  string

	TrxSignReq int
}

var chainContext *ChainContext

func InitContext() {
	chainContext = &ChainContext{}
	chainContext.Groups = make(map[string]*GroupItem)
	chainContext.Version = "ver 0.01"
	//test only, request at least 1 witness signature
	chainContext.TrxSignReq = 1
}

//singlaton
func GetContext() *ChainContext {
	return chainContext
}

func (chainContext *ChainContext) InitDB(datapath string) {
	badgerstorage, err := ds_badger.NewDatastore(datapath+"_bs", nil)
	bs := blockstore.NewBlockstore(ds_sync.MutexWrap(badgerstorage), "bs")

	chainContext.GroupInfoDb, err = badger.Open(badger.DefaultOptions(datapath + "_groups"))
	chainContext.TrxDb, err = badger.Open(badger.DefaultOptions(datapath + "_trx"))
	chainContext.BlockDb, err = badger.Open(badger.DefaultOptions(datapath + "_block"))
	chainContext.BlockStorage = bs

	//defer chainContext.GroupInfoDb.Close()
	//defer chainContext.TrxDb.Close()
	//defer chainContext.BlockDb.Close()

	if err != nil {
		glog.Fatal(err.Error())
	}

	chainContext.DataPath = datapath
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

		if trxMsg.Version != GetContext().Version {
			glog.Infof("Version mismatch")
		} else {
			//glog.Infof("Version ok")
		}

		if trxMsg.Sender == GetContext().PeerId.Pretty() {
			//glog.Infof("Msg from myself, ingore")
			return nil
		}
		
		handleTrxMsg(trxMsg)
	}
}
