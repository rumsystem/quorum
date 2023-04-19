package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	handlers "github.com/rumsystem/quorum/pkg/autorelay/handlers"
)

func (h *RelayServerHandler) Disconnect(c echo.Context) (err error) {
	param := handlers.DelBlacklistParam{}
	if err := c.Bind(&param); err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	result := handlers.DisconnectResult{Ok: true}

	return c.JSON(http.StatusOK, result)
}
