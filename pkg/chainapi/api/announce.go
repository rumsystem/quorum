package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	handlers "github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags User
// @Summary AnnounceUserPubkey
// @Description Announce User's encryption Pubkey to the group
// @Accept json
// @Produce json
// @Param data body handlers.AnnounceParam true "AnnounceParam"
// @Success 200 {object} handlers.AnnounceResult
// @Router /api/v1/group/announce [post]
func (h *Handler) Announce(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(handlers.AnnounceParam)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	sudo := false
	if c.Param("sudo") != "" {
		sudo, err = strconv.ParseBool(c.Param("sudo"))
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}
	}

	res, err := handlers.AnnounceHandler(params, sudo)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}
