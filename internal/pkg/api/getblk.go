package api

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func (h *Handler) GetBlock(c echo.Context) (err error) {
	output := make(map[string]string)
	output[ERROR_INFO] = "Not implement yet"
	return c.JSON(http.StatusOK, output)
}
