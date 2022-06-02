package nodesdk

import (
	"context"

	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	nodesdkdb "github.com/rumsystem/quorum/pkg/nodesdk/db"
	http_client "github.com/rumsystem/quorum/pkg/nodesdk/http"
)

type NodeSdkCtx struct {
	Ctx         context.Context
	Keystore    localcrypto.Keystore
	DbMgr       *nodesdkdb.DbMgr
	HttpClients map[string]*http_client.HttpClient
	Name        string
	Version     string
	PeerId      peer.ID
	PublicKey   p2pcrypto.PubKey
}

var nodesdkCtx *NodeSdkCtx

func GetCtx() *NodeSdkCtx {
	return nodesdkCtx
}

func Init(ctx context.Context, name string, db *nodesdkdb.DbMgr, ver string) {
	nodesdkCtx = &NodeSdkCtx{}
	nodesdkCtx.Name = name
	nodesdkCtx.Ctx = ctx
	nodesdkCtx.Version = ver
	nodesdkCtx.DbMgr = db
	nodesdkCtx.HttpClients = make(map[string]*http_client.HttpClient)
}

func GetDbMgr() *nodesdkdb.DbMgr {
	return nodesdkCtx.DbMgr
}

func GetKeyStore() localcrypto.Keystore {
	return nodesdkCtx.Keystore
}

func (ctx *NodeSdkCtx) GetNextNouce(groupId string, prefix ...string) (nonce uint64, err error) {
	n, err := ctx.DbMgr.GetNextNouce(groupId)
	return n, err
}

func (ctx *NodeSdkCtx) GetHttpClient(groupId string) (*http_client.HttpClient, error) {
	if _, ok := ctx.HttpClients[groupId]; !ok {
		var client *http_client.HttpClient
		client = &http_client.HttpClient{}
		err := client.Init()
		if err != nil {
			return nil, err
		}
		ctx.HttpClients[groupId] = client
	}

	c, _ := ctx.HttpClients[groupId]
	return c, nil
}
