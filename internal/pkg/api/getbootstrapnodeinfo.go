package api

import (
	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"net/http"
)

func (h *Handler) GetBootStropNodeInfo(c echo.Context) (err error) {
	output := make(map[string]interface{})
	output[NODE_STATUS] = "NODE_ONLINE"
	output[NODETYPE] = "bootstrap"
	output[NODE_ID] = nodectx.GetNodeCtx().PeerId.Pretty()
	return c.JSON(http.StatusOK, output)
}
