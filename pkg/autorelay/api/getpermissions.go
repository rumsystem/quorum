package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	handlers "github.com/rumsystem/quorum/pkg/autorelay/handlers"
)

func (h *RelayServerHandler) GetPermissions(c echo.Context) (err error) {
	peer := c.QueryParam("peer")
	if peer == "" {
		return rumerrors.NewBadRequestError("peer can't be nil.")
	}

	result, err := handlers.GetPermissions(h.db, peer)
	if err != nil {
		return rumerrors.NewInternalServerError(err)
	}

	return c.JSON(http.StatusOK, result)
}
