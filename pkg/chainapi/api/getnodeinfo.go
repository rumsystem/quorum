package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
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
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, info)
}
