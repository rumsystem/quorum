package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	_ "github.com/rumsystem/quorum/internal/pkg/stats" // for swag find stats.NetworkStatsSummary
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
	output := make(map[string]string)

	start := c.QueryParam("start")
	end := c.QueryParam("end")
	var startTime *time.Time
	var endTime *time.Time

	if start != "" {
		t, err := time.Parse(time.RFC3339, start)
		if err != nil {
			output[ERROR_INFO] = fmt.Sprintf("parse start query param failed: %s", err)
			return c.JSON(http.StatusBadRequest, output)
		}
		startTime = &t
	}
	if end != "" {
		t, err := time.Parse(time.RFC3339, end)
		if err != nil {
			output[ERROR_INFO] = fmt.Sprintf("parse end query param failed: %s", err)
			return c.JSON(http.StatusBadRequest, output)
		}
		endTime = &t
	}

	result, err := handlers.GetNetworkStats(startTime, endTime)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusInternalServerError, output)
	}

	return c.JSON(http.StatusOK, result)
}
