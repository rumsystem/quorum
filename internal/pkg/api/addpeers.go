package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

// @Tags Node
// @Summary AddPeers
// @Description Connect to peers
// @Accept json
// @Produce json
// @Param data body []string true "Peers List"
// @Success 200 {object} handlers.AddPeerResult
// @Router /api/v1/network/peers [post]
func (h *Handler) AddPeers(c echo.Context) (err error) {
	var input handlers.AddPeerParam
	output := make(map[string]string)

	if err = c.Bind(&input); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	result, err := handlers.AddPeers(input)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}
	return c.JSON(http.StatusOK, result)
}
