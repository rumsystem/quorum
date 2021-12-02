package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
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
	params := new(handlers.LeaveGroupParam)

	if err := c.Bind(params); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}

	res, err := handlers.LeaveGroup(params, h.Appdb)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, res)
}
