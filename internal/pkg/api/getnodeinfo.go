package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

// @Tags Node
// @Summary GetNodeInfo
// @Description Return the node info
// @Produce json
// @Success 200 {object} handlers.NodeInfo
// @Router /api/v1/node [get]
func (h *Handler) GetNodeInfo(c echo.Context) (err error) {
	info, err := handlers.GetNodeInfo(h.Node.NetworkName)
	if err != nil {
		output := make(map[string]interface{})
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, info)
}
