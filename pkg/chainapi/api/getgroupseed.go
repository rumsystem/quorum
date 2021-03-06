package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags Groups
// @Summary Get group seed
// @Description get group seed from appdb
// @Produce json
// @Param group_id path string  true "Group Id"
// @Success 200 {object} handlers.GetGroupSeedResult
// @Router /api/v1/group/{group_id}/seed [get]
func (h *Handler) GetGroupSeedHandler(c echo.Context) (err error) {
	groupId := c.Param("group_id")
	if groupId == "" {
		return rumerrors.NewBadRequestError(rumerrors.ErrInvalidGroupID)
	}

	seed, err := handlers.GetGroupSeed(groupId, h.Appdb)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}
	seedurl, err := handlers.GroupSeedToUrl(1, []string{}, seed)
	if err != nil {
		return rumerrors.NewInternalServerError(fmt.Sprintf("seedurl output failed: %s", err))
	}

	result := handlers.GetGroupSeedResult{
		Seed: seedurl,
	}
	return c.JSON(http.StatusOK, result)
}
