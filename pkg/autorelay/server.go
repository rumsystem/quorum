package autorelay

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
	"github.com/rumsystem/quorum/pkg/autorelay/api"
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
func StartRelayServer(config cli.RelayNodeFlag, quitCh chan os.Signal, h *api.RelayServerHandler) {
	e := utils.NewEcho(config.IsDebug)
	r := e.Group("/relay")

	r.GET("/quit", func(c echo.Context) (err error) {
		fmt.Println("/api/quit has been called, send Signal SIGTERM...")
		quitCh <- syscall.SIGTERM
		return nil
	})

	r.POST("/v1/forbid", h.ForbidPeer)
	r.POST("/v1/blacklist", h.AddBlacklist)
	r.DELETE("/v1/blacklist", h.DeleteBlacklist)
	r.POST("/v1/disconnect", h.Disconnect)

	r.GET("/v1/permissions", h.GetPermissions)
	r.GET("/v1/blacklist", h.GetBlacklist)

	e.Logger.Fatal(e.Start(fmt.Sprintf("%s:%d", config.APIHost, config.APIPort)))
}
