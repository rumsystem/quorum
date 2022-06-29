package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags Groups
// @Summary GetPubQueue
// @Description Return items in the publish queue
// @Produce json
// @Param group_id path string  true "Group Id"
// @Success 200 {object} handlers.PubQueueInfo
// @Router /api/v1/group/{group_id}/pubqueue [get]
func (h *Handler) GetPubQueue(c echo.Context) (err error) {
	groupId := c.Param("group_id")
	trxId := c.QueryParam("trx")
	status := c.QueryParam("status")
	if groupId == "" {
		return rumerrors.NewBadRequestError(rumerrors.ErrInvalidGroupID)
	}

	info, err := handlers.GetPubQueue(groupId, status, trxId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
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
// @Param data body PubQueueAckPayload true "ackpayload"
// @Success 200 {object} []string
// @Router /api/v1/trx/ack [post]
func (h *Handler) PubQueueAck(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)

	payload := &PubQueueAckPayload{}
	if err := cc.BindAndValidate(payload); err != nil {
		return err
	}

	if len(payload.TrxIds) == 0 {
		return rumerrors.NewBadRequestError(rumerrors.ErrInvalidTrxIDList)
	}

	res, err := handlers.PubQueueAck(payload.TrxIds)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}
