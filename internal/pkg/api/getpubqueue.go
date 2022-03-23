package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

// @Tags Groups
// @Summary GetPubQueue
// @Description Return items in the publish queue
// @Produce json
// @Success 200 {object} handlers.PubQueueInfo
// @Router /api/v1/node [get]
func (h *Handler) GetPubQueue(c echo.Context) (err error) {
	output := make(map[string]string)
	groupId := c.Param("group_id")
	if groupId == "" {
		output[ERROR_INFO] = "group_id can't be nil."
		return c.JSON(http.StatusBadRequest, output)
	}

	info, err := handlers.GetPubQueue(groupId)
	if err != nil {
		output := make(map[string]interface{})
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, info)
}
