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
// @Router /api/v1/group/:group_id/pubqueue [get]
func (h *Handler) GetPubQueue(c echo.Context) (err error) {
	output := make(map[string]string)
	groupId := c.Param("group_id")
	trxId := c.QueryParam("trx")
	status := c.QueryParam("status")
	if groupId == "" {
		output[ERROR_INFO] = "group_id can't be nil."
		return c.JSON(http.StatusBadRequest, output)
	}

	info, err := handlers.GetPubQueue(groupId, status, trxId)
	if err != nil {
		output := make(map[string]interface{})
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, info)
}

type PubQueueAckPayload struct {
	TrxIds []string `json:"trx_ids"`
}

// @Tags Chain
// @Summary PubQueueAck
// @Description ack pubqueue trxs
// @Accept json
// @Produce json
// @Param data body PubQueueAckPayload
// @Success 200 {object} []string
// @Router /api/v1/trx/ack [post]
func (h *Handler) PubQueueAck(c echo.Context) (err error) {
	output := make(map[string]string)
	payload := &PubQueueAckPayload{}
	if err = c.Bind(payload); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if len(payload.TrxIds) == 0 {
		output[ERROR_INFO] = "trx_ids is empty"
		return c.JSON(http.StatusBadRequest, output)
	}

	res, err := handlers.PubQueueAck(payload.TrxIds)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}
	return c.JSON(http.StatusOK, res)
}
