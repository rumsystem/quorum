package api

import (
	"github.com/huo-ju/quorum/internal/pkg/chain"
	"github.com/labstack/echo/v4"
	"net/http"
)

func (h *Handler) GetBootStropNodeInfo(c echo.Context) (err error) {
	output := make(map[string]interface{})
	output[NODE_STATUS] = "NODE_ONLINE"
	output[NODETYPE] = "bootstrap"
	output[NODE_ID] = chain.GetNodeCtx().PeerId.Pretty()
	return c.JSON(http.StatusOK, output)
}
