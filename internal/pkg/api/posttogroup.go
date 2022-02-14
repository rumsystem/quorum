package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

// @Tags Groups
// @Summary PostToGroup
// @Description Post object to a group
// @Accept json
// @Produce json
// @Param data body quorumpb.Activity true "Activity object"
// @Success 200 {object} handlers.TrxResult
// @Router /api/v1/group/content [post]
func (h *Handler) PostToGroup(c echo.Context) (err error) {

	output := make(map[string]string)
	paramspb := new(quorumpb.Activity)
	if err = c.Bind(paramspb); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	res, err := handlers.PostToGroup(paramspb)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}
	return c.JSON(http.StatusOK, res)
}
