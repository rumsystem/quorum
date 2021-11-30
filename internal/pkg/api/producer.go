package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

// @Tags Management
// @Summary AddProducer
// @Description add a peer to the group producer list
// @Accept json
// @Produce json
// @Param data body handlers.GrpProducerParam true "GrpProducerParam"
// @Success 200 {object} handlers.GrpProducerResult
// @Router /api/v1/group/producer [post]
func (h *Handler) GroupProducer(c echo.Context) (err error) {
	output := make(map[string]string)
	params := new(handlers.GrpProducerParam)

	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	res, err := handlers.GroupProducer(params)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, res)
}
