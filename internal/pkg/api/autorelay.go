package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

// @Tags Node
// @Summary AddRelayServers
// @Accept json
// @Produce json
// @Param data body []string true "Peers List"
// @Success 200 {boolean}
// @Router /api/v1/network/relay [post]
func (h *Handler) AddRelayServers(c echo.Context) (err error) {
	var input handlers.AddRelayParam
	output := make(map[string]interface{})

	if err = c.Bind(&input); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	ok, err := handlers.AddRelayServers(input)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}
	return c.JSON(http.StatusOK, ok)
}
