package relayapi

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/appdata"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	"github.com/rumsystem/quorum/internal/pkg/utils"
)

type Handler struct {
	Ctx       context.Context
	Appdb     *appdata.AppDb
	Trxdb     def.TrxStorageIface
	Apiroot   string
	GitCommit string
	ConfigDir string
	PeerName  string
	NodeName  string
}

//StartRelayServer : Start local web server
func StartRelayServer(config cli.RelayNodeConfig, quitCh chan os.Signal, h *RelayServerHandler) {
	e := utils.NewEcho(config.IsDebug)
	r := e.Group("/relay")

	r.GET("/quit", func(c echo.Context) (err error) {
		fmt.Println("/api/quit has been called, send Signal SIGTERM...")
		quitCh <- syscall.SIGTERM
		return nil
	})

	r.POST("/v1/peer/forbid", h.ForbidPeer)

	r.GET("/v1/peer/permissions", h.GetPermissions)

	certPath, keyPath, err := utils.GetTLSCerts()
	if err != nil {
		panic(err)
	}
	e.Logger.Fatal(e.StartTLS(config.APIListenAddresses, certPath, keyPath))
}
