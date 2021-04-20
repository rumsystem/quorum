package api

import (
	//"fmt"
	"net/http"
	//"encoding/json"
	//kb "github.com/libp2p/go-libp2p-kbucket"
	"github.com/labstack/echo/v4"
)

func (h *Handler) GetTrx(c echo.Context) (err error) {
	output := make(map[string]string)
	output[ERROR_INFO] = "Not implement yet"
	return c.JSON(http.StatusOK, output)
}
