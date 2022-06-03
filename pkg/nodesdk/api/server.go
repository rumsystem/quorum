package nodesdkapi

import (
	"fmt"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/appapi"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/encoding/protojson"
)

var quitch chan os.Signal

//StartAPIServer : Start local web server
func StartNodeSDKServer(config cli.Config, signalch chan os.Signal, h *NodeSDKHandler, nodeopt *options.NodeOptions) {
	quitch = signalch
	e := echo.New()
	e.Binder = new(CustomBinder)
	e.Use(middleware.JWTWithConfig(appapi.CustomJWTConfig(nodeopt.JWTKey)))
	r := e.Group("/nodesdk_api")

	r.GET("/quit", quitapp)
	r.POST("/v1/group/join", h.JoinGroup())
	r.POST("/v1/group/leave", h.LeaveGroup())
	r.POST("/v1/group/content", h.PostToGroup())
	r.POST("/v1/group/getctn", h.GetGroupCtn())
	r.POST("/v1/group/profile", h.UpdProfile)
	r.POST("/v1/group/apihosts", h.UpdApiHostUrl)
	r.POST("/v1/keystore/create", h.CreateNewKeyWithAlias())
	r.POST("/v1/keystore/bindalias", h.BindAliasWithKeyName())
	r.POST("/v1/keystore/remove", h.RmAlias())

	r.GET("/v1/group/listall", h.GetAllGroups())
	r.GET("/v1/group/:group_id/list", h.GetGroupById())
	r.GET("/v1/group/:group_id/seed", h.GetGroupSeed())
	r.GET("/v1/keystore/listall", h.GetAllAlias())
	r.GET("/v1/trx/:group_id/:trx_id", h.GetTrx())
	r.GET("/v1/block/:group_id/:block_id", h.GetBlock())
	r.GET("/v1/group/:group_id/info", h.GetGroupInfo)
	r.GET("/v1/group/:group_id/producers", h.GetProducers)
	r.GET("/v1/group/:group_id/announced/users", h.GetAnnouncedUsers)
	r.GET("/v1/group/:group_id/announced/user/:sign_pubkey", h.GetAnnouncedUsers)
	r.GET("/v1/group/:group_id/appconfig/keylist", h.GetAppConfigKey)
	r.GET("/v1/group/:group_id/appconfig/:key", h.GetAppConfigItem)

	//not support, chainSdk should give something else to the nodesdk
	//r.GET("/v1/node", h.GetNodeInfo)

	//not support, should not return this to nodesdk
	//r.POST("/v1/group/announce", h.Announce)
	//r.GET("/v1/group/:group_id/trx/allowlist", h.GetChainTrxAllowList)
	//r.GET("/v1/group/:group_id/trx/denylist", h.GetChainTrxDenyList)
	//r.GET("/v1/group/:group_id/trx/auth/:trx_type", h.GetChainTrxAuthMode)

	certPath, keyPath, err := utils.GetTLSCerts()
	if err != nil {
		panic(err)
	}
	e.Logger.Fatal(e.StartTLS(config.NodeAPIListenAddress, certPath, keyPath))
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
