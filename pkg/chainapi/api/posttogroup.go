package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags Groups
// @Summary PostToGroup
// @Description Post object to a group
// @Accept json
// @Produce json
// @Param group_id path string  true "Group Id"
// @Param data body handlers.PostToGroupParam true "payload"
// @Success 200 {object} handlers.TrxResult
// @Router /api/v1/group/{group_id}/content [post]
func (h *Handler) PostToGroup(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	payload := handlers.PostToGroupParam{}
	if err := cc.BindAndValidate(&payload); err != nil {
		return err
	}

	res, err := handlers.PostToGroup(&payload, payload.Sudo)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}
