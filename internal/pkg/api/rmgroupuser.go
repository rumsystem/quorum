package api

import (
	"github.com/labstack/echo/v4"
	//"io/ioutil"
	"net/http"
)

func (h *Handler) RmGroupUser(c echo.Context) (err error) {
	//refer post.go
	output := make(map[string]string)
	return c.JSON(http.StatusOK, output)
}
