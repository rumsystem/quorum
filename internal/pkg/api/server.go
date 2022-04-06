package api

import (
	"fmt"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/p2p"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	appapi "github.com/rumsystem/quorum/pkg/app/api"
	"google.golang.org/protobuf/encoding/protojson"
)

var quitch chan os.Signal

//StartAPIServer : Start local web server
func StartAPIServer(config cli.Config, signalch chan os.Signal, h *Handler, apph *appapi.Handler, node *p2p.Node, nodeopt *options.NodeOptions, ks localcrypto.Keystore, ethaddr string, isbootstrapnode bool) {
	quitch = signalch
	e := echo.New()
	e.Binder = new(CustomBinder)
	e.Use(middleware.JWTWithConfig(appapi.CustomJWTConfig(nodeopt.JWTKey)))
	r := e.Group("/api")
	a := e.Group("/app/api")
	r.GET("/quit", quitapp)
	if isbootstrapnode == false {
		r.POST("/v1/group", h.CreateGroup())
		r.POST("/v1/group/join", h.JoinGroup())
		r.POST("/v1/group/leave", h.LeaveGroup)
		r.POST("/v1/group/clear", h.ClearGroupData)
		r.POST("/v1/group/content", h.PostToGroup)
		r.POST("/v1/group/profile", h.UpdateProfile)
		r.POST("/v1/network/peers", h.AddPeers)
		r.POST("/v1/group/chainconfig", h.MgrChainConfig)
		r.POST("/v1/group/producer", h.GroupProducer)
		r.POST("/v1/group/user", h.GroupUser)
		r.POST("/v1/group/announce", h.Announce)
		//r.POST("/v1/group/schema", h.Schema)
		r.POST("/v1/group/:group_id/startsync", h.StartSync)
		r.POST("/v1/group/appconfig", h.MgrAppConfig)
		r.GET("/v1/node", h.GetNodeInfo)
		r.POST("/v1/rex/initsession", h.RexInitSession(node))
		r.GET("/v1/network", h.GetNetwork(&node.Host, node.Info, nodeopt, ethaddr))
		r.GET("/v1/network/peers/ping", h.PingPeer(node))
		r.POST("/v1/psping", h.PSPingPeer(node))
		r.GET("/v1/block/:group_id/:block_id", h.GetBlockById)
		r.GET("/v1/trx/:group_id/:trx_id", h.GetTrx)
		r.POST("/v1/trx/ack", h.PubQueueAck)

		r.GET("/v1/groups", h.GetGroups)
		r.GET("/v1/group/:group_id/content", h.GetGroupCtn)
		r.GET("/v1/group/:group_id/trx/allowlist", h.GetChainTrxAllowList)
		r.GET("/v1/group/:group_id/trx/denylist", h.GetChainTrxDenyList)
		r.GET("/v1/group/:group_id/trx/auth/:trx_type", h.GetChainTrxAuthMode)
		r.GET("/v1/group/:group_id/producers", h.GetGroupProducers)
		r.GET("/v1/group/:group_id/announced/users", h.GetAnnouncedGroupUsers)
		r.GET("/v1/group/:group_id/announced/user/:sign_pubkey", h.GetAnnouncedGroupUser)
		r.GET("/v1/group/:group_id/announced/producers", h.GetAnnouncedGroupProducer)
		//r.GET("/v1/group/:group_id/app/schema", h.GetGroupAppSchema)
		r.GET("/v1/group/:group_id/appconfig/keylist", h.GetAppConfigKey)
		r.GET("/v1/group/:group_id/appconfig/:key", h.GetAppConfigItem)
		r.GET("/v1/group/:group_id/seed", h.GetGroupSeedHandler)
		r.GET("/v1/group/:group_id/pubqueue", h.GetPubQueue)

		a.POST("/v1/group/:group_id/content", apph.ContentByPeers)
		a.POST("/v1/token/apply", apph.ApplyToken)
		a.POST("/v1/token/refresh", apph.RefreshToken)
	} else {
		r.GET("/v1/node", h.GetBootstrapNodeInfo)
	}

	certPath, keyPath, err := utils.GetTLSCerts()
	if err != nil {
		panic(err)
	}
	e.Logger.Fatal(e.StartTLS(config.APIListenAddresses, certPath, keyPath))
}

type CustomBinder struct{}

func (cb *CustomBinder) Bind(i interface{}, c echo.Context) (err error) {
	db := new(echo.DefaultBinder)
	switch i.(type) {
	case *quorumpb.Activity:
		bodyBytes, err := ioutil.ReadAll(c.Request().Body)
		err = protojson.Unmarshal(bodyBytes, i.(*quorumpb.Activity))
		return err
	default:
		if err = db.Bind(i, c); err != echo.ErrUnsupportedMediaType {
			return
		}
		return err
	}
}

func quitapp(c echo.Context) (err error) {
	fmt.Println("/api/quit has been called, send Signal SIGTERM...")
	quitch <- syscall.SIGTERM
	return nil
}
