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
// @Description Leave a new group
// @Accept json
// @Produce json
// @Param data body handlers.LeaveGroupParam true "LeaveGroupParam"
// @success 200 {object} handlers.LeaveGroupResult "LeaveGroupResult"
// @Router /api/v1/group/leave [post]
func (h *Handler) LeaveGroup(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(handlers.LeaveGroupParam)

	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	res, err := handlers.LeaveGroup(params, h.Appdb)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}
