package nodesdk_api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (h *NodeSDKHandler) GetNodeInfo(c echo.Context) (err error) {
	return c.JSON(http.StatusOK, nil)
}
