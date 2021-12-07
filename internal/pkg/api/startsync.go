package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

type StartSyncResult struct {
	GroupId string `validate:"required"`
	Error   string
}

// @Tags Group
// @Summary StartSync
// @Description Start sync
// @Produce json
// @Success 200 {object} StartSyncResult
// @Router /api/v1/group/{group_id}/startsync [post]
func (h *Handler) StartSync(c echo.Context) (err error) {
	output := make(map[string]string)
	groupid := c.Param("group_id")

	res, err := handlers.StartSync(groupid)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}
	return c.JSON(http.StatusOK, res)
}
