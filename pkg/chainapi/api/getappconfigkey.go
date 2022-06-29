package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags Management
// @Summary GetAppConfigKey
// @Description get app config key list
// @Produce json
// @Param group_id path string  true "Group Id"
// @Success 200 {array} handlers.AppConfigKeyListItem
// @Router /api/v1/group/{group_id}/config/keylist [get]
func (h *Handler) GetAppConfigKey(c echo.Context) (err error) {
	groupid := c.Param("group_id")
	res, err := handlers.GetAppConfigKeyList(groupid)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}
