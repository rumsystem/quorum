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
	customJWTConfig := appapi.CustomJWTConfig(nodeopt.JWTKey)
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

func StartProducerServer(config StartServerParam, signalch chan os.Signal, h *Handler, node *p2p.Node, nodeopt *options.NodeOptions, ks localcrypto.Keystore, ethaddr string) {
	quitch = signalch
	e := utils.NewEcho(config.IsDebug)
	customJWTConfig := appapi.CustomJWTConfig(nodeopt.JWTKey)
	e.Use(middleware.JWTWithConfig(customJWTConfig))
	e.Use(rummiddleware.OpaWithConfig(rummiddleware.OpaConfig{
		Skipper:   rummiddleware.LocalhostSkipper,
		Policy:    policyStr,
		Query:     "x = data.quorum.restapi.authz.allow", // FIXME: hardcode
		InputFunc: opaInputFunc,
	}))
	r := e.Group("/api")
	r.GET("/quit", quitapp)

	//r.POST("/v1/group", h.CreateGroupUrl())
	//r.POST("/v1/group/join", h.JoinGroup())
	r.POST("/v2/group/join", h.JoinGroupV2())
	r.POST("/v1/group/leave", h.LeaveGroup)
	r.POST("/v1/group/clear", h.ClearGroupData)
	r.POST("/v1/group/announce", h.Announce)

	r.GET("/v1/node", h.GetNodeInfo)
	r.GET("/v1/network", h.GetNetwork(&node.Host, node.Info, nodeopt, ethaddr))
	//r.GET("/v1/network/stats", h.GetNetworkStatsSummary)
	r.GET("/v1/block/:group_id/:block_id", h.GetBlock)
	r.GET("/v1/trx/:group_id/:trx_id", h.GetTrx)

	r.GET("/v1/groups", h.GetGroups)
	r.GET("/v1/group/:group_id/trx/allowlist", h.GetChainTrxAllowList)
	r.GET("/v1/group/:group_id/trx/denylist", h.GetChainTrxDenyList)
	r.GET("/v1/group/:group_id/trx/auth/:trx_type", h.GetChainTrxAuthMode)
	r.GET("/v1/group/:group_id/producers", h.GetGroupProducers)
	r.GET("/v1/group/:group_id/announced/users", h.GetAnnouncedGroupUsers)
	r.GET("/v1/group/:group_id/announced/user/:sign_pubkey", h.GetAnnouncedGroupUser)
	r.GET("/v1/group/:group_id/announced/producers", h.GetAnnouncedGroupProducer)
	r.GET("/v1/group/:group_id/seed", h.GetGroupSeedHandler)

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
func StartFullNodeServer(config StartServerParam, signalch chan os.Signal, h *Handler, apph *appapi.Handler, node *p2p.Node, nodeopt *options.NodeOptions, ks localcrypto.Keystore, ethaddr string) {
	quitch = signalch
	e := utils.NewEcho(config.IsDebug)
	customJWTConfig := appapi.CustomJWTConfig(nodeopt.JWTKey)
	e.Use(middleware.JWTWithConfig(customJWTConfig))
	e.Use(rummiddleware.OpaWithConfig(rummiddleware.OpaConfig{
		Skipper:   rummiddleware.LocalhostSkipper,
		Policy:    policyStr,
		Query:     "x = data.quorum.restapi.authz.allow", // FIXME: hardcode
		InputFunc: opaInputFunc,
	}))

	// prometheus metric
	e.GET("/metrics", h.Metrics)

	r := e.Group("/api")
	a := e.Group("/app/api")
	r.GET("/quit", quitapp)

	//POST API does not support sudo
	r.POST("/v1/group", h.CreateGroupUrl())
	r.POST("/v2/group/join", h.JoinGroupV2())
	r.POST("/v1/group/leave", h.LeaveGroup)
	r.POST("/v1/group/clear", h.ClearGroupData)
	r.POST("/v1/network/peers", h.AddPeers)
	r.POST("/v1/group/:group_id/startsync", h.StartSync)
	//r.POST("/v1/psping", h.PSPingPeer(node))
	//r.POST("/v1/ping", h.P2PPingPeer(node))
	r.POST("/v1/tools/pubkeytoaddr", h.PubkeyToEthaddr)
	r.POST("/v1/tools/seedurlextend", h.SeedUrlextend)
	r.POST("/v1/trx/ack", h.PubQueueAck)
	r.POST("/v1/group/reqpsync", h.ReqPSync)
	//r.POST("/v1/group/join", h.JoinGroup())

	r.POST("/v1/group/:group_id/content", h.PostToGroup)

	r.POST("/v1/group/appconfig", h.MgrAppConfig)
	r.POST("/v1/group/appconfig/:sudo", h.MgrAppConfig)

	r.POST("/v1/group/chainconfig", h.MgrChainConfig)
	r.POST("/v1/group/chainconfig/:sudo", h.MgrChainConfig)

	r.POST("/v1/group/producer", h.GroupProducer)
	r.POST("/v1/group/producer/:sudo", h.GroupProducer)

	r.POST("/v1/group/user", h.GroupUser)
	r.POST("/v1/group/user/:sudo", h.GroupUser)

	r.POST("/v1/group/announce", h.Announce)
	r.POST("/v1/group/announce/:sudo", h.Announce)

	r.GET("/v1/node", h.GetNodeInfo)
	r.GET("/v1/network", h.GetNetwork(&node.Host, node.Info, nodeopt, ethaddr))
	//r.GET("/v1/network/stats", h.GetNetworkStatsSummary)
	//r.GET("/v1/network/peers/ping", h.PingPeers(node))
	r.GET("/v1/block/:group_id/:block_id", h.GetBlock)
	r.GET("/v1/trx/:group_id/:trx_id", h.GetTrx)
	r.GET("/v1/groups", h.GetGroups)
	r.GET("/v1/group/:group_id/trx/allowlist", h.GetChainTrxAllowList)
	r.GET("/v1/group/:group_id/trx/denylist", h.GetChainTrxDenyList)
	r.GET("/v1/group/:group_id/trx/auth/:trx_type", h.GetChainTrxAuthMode)
	r.GET("/v1/group/:group_id/producers", h.GetGroupProducers)
	r.GET("/v1/group/:group_id/announced/users", h.GetAnnouncedGroupUsers)
	r.GET("/v1/group/:group_id/announced/user/:sign_pubkey", h.GetAnnouncedGroupUser)
	r.GET("/v1/group/:group_id/announced/producers", h.GetAnnouncedGroupProducer)
	r.GET("/v1/group/:group_id/appconfig/keylist", h.GetAppConfigKey)
	r.GET("/v1/group/:group_id/appconfig/:key", h.GetAppConfigItem)
	r.GET("/v1/group/:group_id/seed", h.GetGroupSeedHandler)
	r.GET("/v1/group/:group_id/pubqueue", h.GetPubQueue)

	//app api
	a.POST("/v1/token/refresh", apph.RefreshToken)
	a.POST("/v1/token/create", apph.CreateToken)
	a.GET("/v1/group/:group_id/content", apph.ContentByPeers)

	if nodeopt.EnableRelay {
		r.POST("/v1/network/relay", h.AddRelayServers)
	}

	//utils
	r.POST("/v1/keystore/signtx", h.SignTx)

	//for nodesdk
	r.POST("/v1/node/trx/:group_id", h.SendTrx)
	r.POST("/v1/node/groupctn/:group_id", h.GetContentNSdk)
	r.POST("/v1/node/getchaindata/:group_id", h.GetDataNSdk)
	r.GET("/v1/node/getencryptpubkeys/:group_id", h.GetUserEncryptPubKeys)
	r.POST("/v1/node/announce/:group_id", h.AnnounceNodeSDK)

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
