package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
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
	output := make(map[string]string)
	params := new(handlers.AnnounceParam)

	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	res, err := handlers.AnnounceHandler(params)

	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, res)
}
