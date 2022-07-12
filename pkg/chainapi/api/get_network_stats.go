package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors" // for swag find stats.NetworkStatsSummary
	_ "github.com/rumsystem/quorum/internal/pkg/stats"          // for swag find stats.NetworkStatsSummary
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags Node
// @Summary GetNetworkStatsSummary
// @Description Get network stats summary
// @Produce json
// @Param start query time.Time false "Start Time"
// @Param end query time.Time false "End Time"
// @Success 200 {object} stats.NetworkStatsSummary
// @Router /api/v1/network/stats [get]
func (h *Handler) GetNetworkStatsSummary(c echo.Context) error {
	start := c.QueryParam("start")
	end := c.QueryParam("end")
	var startTime *time.Time
	var endTime *time.Time

	if start != "" {
		t, err := time.Parse(time.RFC3339, start)
		if err != nil {
			return rumerrors.NewBadRequestError(fmt.Sprintf("parse start query param failed: %s", err))
		}
		startTime = &t
	}
	if end != "" {
		t, err := time.Parse(time.RFC3339, end)
		if err != nil {
			return rumerrors.NewBadRequestError(fmt.Sprintf("parse end query param failed: %s", err))
		}
		endTime = &t
	}

	result, err := handlers.GetNetworkStats(startTime, endTime)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}
