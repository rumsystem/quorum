package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	handlers "github.com/rumsystem/quorum/pkg/autorelay/handlers"
)

func (h *RelayServerHandler) ForbidPeer(c echo.Context) (err error) {
	param := handlers.ForbidParam{}
	if err := c.Bind(&param); err != nil {
		rumerrors.NewBadRequestError(err.Error())
	}

	result, err := handlers.ForbidPeer(h.db, param)
	if err != nil {
		return rumerrors.NewInternalServerError(err)
	}

	return c.JSON(http.StatusOK, result)
}
