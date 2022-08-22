package nodesdk

import (
	"context"

	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	chainstorage "github.com/rumsystem/quorum/internal/pkg/storage/chain"

	//nodesdkdb "github.com/rumsystem/quorum/pkg/nodesdk/db"
	http_client "github.com/rumsystem/quorum/pkg/nodesdk/http"
)

type NodeSdkCtx struct {
	Ctx         context.Context
	Keystore    localcrypto.Keystore
	HttpClients map[string]*http_client.HttpClient
	Name        string
	Version     string
	PeerId      peer.ID
	PublicKey   p2pcrypto.PubKey
	chaindb     *chainstorage.Storage
}

var dbMgr *storage.DbMgr

var nodesdkCtx *NodeSdkCtx

func GetCtx() *NodeSdkCtx {
	return nodesdkCtx
}

func Init(ctx context.Context, name string, db *storage.DbMgr, chaindb *chainstorage.Storage) {
	nodesdkCtx = &NodeSdkCtx{}
	nodesdkCtx.Name = name
	nodesdkCtx.Ctx = ctx
	nodesdkCtx.Version = "1.0.0"
	nodesdkCtx.chaindb = chaindb
	nodesdkCtx.HttpClients = make(map[string]*http_client.HttpClient)
	dbMgr = db
}

func GetDbMgr() *storage.DbMgr {
	return dbMgr
}

func GetKeyStore() localcrypto.Keystore {
	return nodesdkCtx.Keystore
}

func (ctx *NodeSdkCtx) GetChainStorage() *chainstorage.Storage {
	return nodesdkCtx.chaindb
}

func (ctx *NodeSdkCtx) GetNextNouce(groupId string, prefix ...string) (nonce uint64, err error) {
	n, err := dbMgr.GetNextNouce(groupId)
	return n, err
}

func (ctx *NodeSdkCtx) GetHttpClient(groupId string) (*http_client.HttpClient, error) {
	if _, ok := ctx.HttpClients[groupId]; !ok {
		client := &http_client.HttpClient{}
		err := client.Init()
		if err != nil {
			return nil, err
		}
		ctx.HttpClients[groupId] = client
	}

	c, _ := ctx.HttpClients[groupId]
	return c, nil
}
