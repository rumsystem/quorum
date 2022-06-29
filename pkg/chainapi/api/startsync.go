package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

type StartSyncResult struct {
	GroupId string `validate:"required"`
	Error   string
}

// @Tags Group
// @Summary StartSync
// @Description Start sync
// @Produce json
// @Param group_id path string  true "Group Id"
// @Success 200 {object} StartSyncResult
// @Router /api/v1/group/{group_id}/startsync [post]
func (h *Handler) StartSync(c echo.Context) (err error) {
	groupid := c.Param("group_id")

	res, err := handlers.StartSync(groupid)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}
