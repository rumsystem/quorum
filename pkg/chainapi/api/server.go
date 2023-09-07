package api

import (
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rumsystem/ip-cert/pkg/zerossl"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p"
	rummiddleware "github.com/rumsystem/quorum/internal/pkg/middleware"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	appapi "github.com/rumsystem/quorum/pkg/chainapi/appapi"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	"golang.org/x/crypto/acme/autocert"
)

var quitch chan os.Signal

type StartServerParam struct {
	IsDebug       bool
	APIHost       string
	APIPort       uint
	CertDir       string
	ZeroAccessKey string
}

// StartAPIServer : Start local web server
func StartBootstrapNodeServer(config StartServerParam, signalch chan os.Signal, h *Handler, apph *appapi.Handler, node *p2p.Node, nodeopt *options.NodeOptions, ks localcrypto.Keystore, ethaddr string) {
	quitch = signalch
	e := utils.NewEcho(config.IsDebug)
	customJWTConfig := appapi.CustomJWTConfig(nodeopt.JWT.Key)
	e.Use(middleware.JWTWithConfig(customJWTConfig))
	e.Use(rummiddleware.OpaWithConfig(rummiddleware.OpaConfig{
		Skipper:   rummiddleware.LocalhostSkipper,
		Policy:    policyStr,
		Query:     "x = data.quorum.restapi.authz.allow", // FIXME: hardcode
		InputFunc: opaInputFunc,
	}))

	r := e.Group("/api")
	r.GET("/quit", quitapp)
	r.GET("/v1/node", h.GetBootstrapNodeInfo)

	// start https or http server
	host := config.APIHost
	if utils.IsDomainName(host) { // domain
		e.AutoTLSManager.Cache = autocert.DirCache(config.CertDir)
		e.AutoTLSManager.HostPolicy = autocert.HostWhitelist(config.APIHost)
		e.AutoTLSManager.Prompt = autocert.AcceptTOS
		e.Logger.Fatal(e.StartAutoTLS(fmt.Sprintf(":%d", config.APIPort)))
	} else if utils.IsPublicIP(host) { // public ip
		ip := net.ParseIP(host)
		privKeyPath, certPath, err := zerossl.IssueIPCert(config.CertDir, ip, config.ZeroAccessKey)
		if err != nil {
			e.Logger.Fatal(err)
		}
		e.Logger.Fatal(e.StartTLS(fmt.Sprintf(":%d", config.APIPort), certPath, privKeyPath))
	} else { // start http server
		e.Logger.Fatal(e.Start(fmt.Sprintf("%s:%d", host, config.APIPort)))
	}
}

// StartAPIServer : Start local web server
func StartRumLiteNodeServer(config StartServerParam, signalch chan os.Signal, h *Handler, apph *appapi.Handler, node *p2p.Node, nodeopt *options.NodeOptions, ks localcrypto.Keystore, ethaddr string) {
	quitch = signalch
	e := utils.NewEcho(config.IsDebug)
	customJWTConfig := appapi.CustomJWTConfig(nodeopt.JWT.Key)
	e.Use(middleware.JWTWithConfig(customJWTConfig))
	e.Use(rummiddleware.OpaWithConfig(rummiddleware.OpaConfig{
		Skipper:   rummiddleware.JWTSkipper,
		Policy:    policyStr,
		Query:     "x = data.quorum.restapi.authz.allow", // FIXME: hardcode
		InputFunc: opaInputFunc,
	}))

	// prometheus metric
	e.GET("/metrics", h.Metrics)

	r := e.Group("/api")
	a := e.Group("/app/api")
	r.GET("/quit", quitapp)

	r.POST("/v2/keystore/createsignkey", h.CreateSignKey())
	r.GET("/v2/keystore/getkeybykeyname", h.GetPubkeyByKeyName())
	r.GET("/v2/keystore/getallkeys", h.GetAllKeys())

	r.POST("/v2/group/newseed", h.NewGroupSeed())
	r.POST("/v2/group/joingroupbyseed", h.JoinGroupBySeed())

	r.POST("/v2/cellar/newseed", h.NewCellarSeed())
	r.POST("/v2/cellar/joincellarbyseed", h.JoinCellarBySeed())

	//r.POST("/v2/cella/leave",h.LeaveCella)
	//r.POST("/v2/cella/clear",h.ClearCellaData)
	//r.GET("/v2/cellas", h.GetCellas)
	//r.GET("/v2/cella/:cella_id", h.GetCellaById)

	r.POST("/v2/group/open", h.OpenGroup)
	r.POST("/v2/group/close", h.CloseGroup)
	r.POST("/v2/group/updsyncer", h.UpdGroupSyncer)

	r.POST("/v1/group/leave", h.LeaveGroup)
	r.POST("/v1/group/clear", h.ClearGroupData)

	r.POST("/v1/network/peers", h.AddPeers)
	r.POST("/v1/tools/pubkeytoaddr", h.PubkeyToEthaddr)
	r.POST("/v1/tools/seedurlextend", h.SeedUrlextend)
	r.POST("/v1/group/:group_id/content", h.PostToGroup)
	r.POST("/v1/group/appconfig", h.MgrAppConfig)
	r.POST("/v1/group/chainconfig", h.MgrChainConfig)

	//get block by id
	r.GET("/v1/block/:group_id/:block_id", h.GetBlock)

	//get trx by id
	r.GET("/v1/trx/:group_id/:trx_id", h.GetTrx)

	//get all groups
	r.GET("/v1/groups", h.GetGroups)

	//get group by id
	r.GET("/v1/group/:group_id", h.GetGroupById)

	//get trx allow list
	r.GET("/v1/group/:group_id/trx/allowlist", h.GetChainTrxAllowList)

	//get trx deny list
	r.GET("/v1/group/:group_id/trx/denylist", h.GetChainTrxDenyList)

	//get trx auth mode
	r.GET("/v1/group/:group_id/trx/auth/:trx_type", h.GetChainTrxAuthMode)

	//get app config key list
	r.GET("/v1/group/:group_id/appconfig/keylist", h.GetAppConfigKey)

	//get app config item
	r.GET("/v1/group/:group_id/appconfig/:key", h.GetAppConfigItem)

	//get group seed
	r.GET("/v1/group/:group_id/seed", h.GetGroupSeedHandler)

	//get node info
	r.GET("/v1/node", h.GetNodeInfo)

	//get group content
	a.GET("/v1/group/:group_id/content", apph.ContentByPeers)

	//get network status
	r.GET("/v1/network", h.GetNetwork(&node.Host, node.Info, nodeopt, ethaddr))

	//app api
	a.POST("/v1/token", apph.CreateToken)
	a.DELETE("/v1/token", apph.RemoveToken)
	a.POST("/v1/token/refresh", apph.RefreshToken)
	a.POST("/v1/token/revoke", apph.RevokeToken)
	a.GET("/v1/token/list", apph.ListToken)

	if nodeopt.EnableRelay {
		r.POST("/v1/network/relay", h.AddRelayServers)
	}

	//utils
	r.POST("/v1/keystore/signtx", h.SignTx)

	// websocket
	r.GET("/v1/ws/trx", h.WebsocketManager.WsConnect)

	//for nodesdk
	{
		n := e.Group("/api/v1/node")

		n.POST("/:group_id/trx", h.NSdkSendTrx)
		n.GET("/:group_id/groupctn", h.GetNSdkContent)

		// auth
		n.GET("/:group_id/auth/by/:trx_type", h.GetNSdkAuthType)
		n.GET("/:group_id/auth/alwlist", h.GetNSdkAllowList)
		n.GET("/:group_id/auth/denylist", h.GetNSdkDenyList)

		// appconfig
		n.GET("/:group_id/appconfig/keylist", h.GetNSdkAppconfigKeylist)
		n.GET("/:group_id/appconfig/by/:key", h.GetNSdkAppconfigByKey)

		n.GET("/:group_id/producers", h.GetNSdkGroupProducers)
		n.GET("/:group_id/info", h.GetNSdkGroupInfo)
		//n.GET("/:group_id/encryptpubkeys", h.GetNSdkUserEncryptPubKeys)
	}

	//deprecated
	//r.POST("/v1/group", h.CreateGroupUrl())
	//r.POST("/v2/group/join", h.JoinGroupV2())
	//r.POST("/v1/group/:group_id/startsync", h.StartSync)

	// start https or http server
	host := config.APIHost
	if utils.IsDomainName(host) { // domain
		e.AutoTLSManager.Cache = autocert.DirCache(config.CertDir)
		e.AutoTLSManager.HostPolicy = autocert.HostWhitelist(config.APIHost)
		e.AutoTLSManager.Prompt = autocert.AcceptTOS
		e.Logger.Fatal(e.StartAutoTLS(fmt.Sprintf(":%d", config.APIPort)))
	} else if utils.IsPublicIP(host) { // public ip
		ip := net.ParseIP(host)
		privKeyPath, certPath, err := zerossl.IssueIPCert(config.CertDir, ip, config.ZeroAccessKey)
		if err != nil {
			e.Logger.Fatal(err)
		}
		e.Logger.Fatal(e.StartTLS(fmt.Sprintf(":%d", config.APIPort), certPath, privKeyPath))
	} else { // start http server
		e.Logger.Fatal(e.Start(fmt.Sprintf("%s:%d", host, config.APIPort)))
	}
}

func quitapp(c echo.Context) (err error) {
	fmt.Println("/api/quit has been called, send Signal SIGTERM...")
	quitch <- syscall.SIGTERM
	return nil
}
