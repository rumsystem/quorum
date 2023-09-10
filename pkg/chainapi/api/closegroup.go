package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags Groups
// @Summary LeaveGroup
// @Description Leave a group
// @Accept json
// @Produce json
// @Param data body handlers.CloseGroupParam true "CloseGroupParam"
// @success 200 {object} handlers.LeaveGroupResult "CloseGroupResult"
// @Router /api/v1/group/close [post]
func (h *Handler) CloseGroup(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(handlers.CloseGroupParam)

	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	res, err := handlers.CloseGroup(params)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}
