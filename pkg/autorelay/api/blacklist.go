package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	handlers "github.com/rumsystem/quorum/pkg/autorelay/handlers"
)

func (h *RelayServerHandler) GetBlacklist(c echo.Context) (err error) {
	peer := c.QueryParam("peer")
	if peer == "" {
		return rumerrors.NewBadRequestError("peer can't be nil.")
	}

	result, err := handlers.GetBlacklist(h.db, peer)
	if err != nil {
		return rumerrors.NewInternalServerError(err)
	}

	return c.JSON(http.StatusOK, result)
}

func (h *RelayServerHandler) AddBlacklist(c echo.Context) (err error) {
	param := handlers.AddBlacklistParam{}
	if err := c.Bind(&param); err != nil {
		rumerrors.NewBadRequestError(err.Error())
	}

	result, err := handlers.AddBlacklist(h.db, param)
	if err != nil {
		return rumerrors.NewInternalServerError(err)
	}

	return c.JSON(http.StatusOK, result)
}

func (h *RelayServerHandler) DeleteBlacklist(c echo.Context) (err error) {
	param := handlers.DelBlacklistParam{}
	if err := c.Bind(&param); err != nil {
		rumerrors.NewBadRequestError(err.Error())
	}

	result, err := handlers.DeleteBlacklist(h.db, param)
	if err != nil {
		return rumerrors.NewInternalServerError(err)
	}

	return c.JSON(http.StatusOK, result)
}
