package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

type SeedUrlextendParam struct {
	SeedURL string `json:"seed" validate:"required"`
}

func (h *Handler) SeedUrlextend() echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := c.(*utils.CustomContext)

		param := new(SeedUrlextendParam)
		if err := cc.BindAndValidate(param); err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		seed, _, err := handlers.UrlToGroupSeed(param.SeedURL)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		return c.JSON(http.StatusOK, seed)
	}
}
