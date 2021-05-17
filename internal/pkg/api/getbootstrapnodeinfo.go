package api

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func (h *Handler) GetBootStropNodeInfo(c echo.Context) (err error) {
	output := make(map[string]string)
	output[NODE_STATUS] = "NODE_ONLINE"
	return c.JSON(http.StatusOK, output)

}
