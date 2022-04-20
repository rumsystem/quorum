package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
)

// @Tags Node
// @Summary GetBootstrapNodeInfo
// @Description Return the bootstrap node info
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/v1/node [get]
func (h *Handler) GetBootstrapNodeInfo(c echo.Context) (err error) {
	output := make(map[string]interface{})
	output[NODE_STATUS] = "NODE_ONLINE"
	output[NODETYPE] = "bootstrap"
	output[NODE_ID] = nodectx.GetNodeCtx().PeerId.Pretty()
	return c.JSON(http.StatusOK, output)
}
