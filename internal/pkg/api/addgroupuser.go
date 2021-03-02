package api

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func (h *Handler) AddGroupUser(c echo.Context) (err error) {
	//refer post.go
	output := make(map[string]string)
	return c.JSON(http.StatusOK, output)
}
