package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/options"
	handlers "github.com/rumsystem/quorum/pkg/chainapi/handlers"
	_ "github.com/rumsystem/rumchaindata/pkg/pb" //import for swaggo
)

// @Tags Groups
// @Summary CreateGroupUrl
// @Description Create a new group
// @Accept json
// @Produce json
// @Param data body handlers.CreateGroupParam true "GroupInfo"
// @Success 200 {object} handlers.GroupSeed
// @Router /api/v1/group [post]
func (h *Handler) CreateGroupUrl() echo.HandlerFunc {
	return func(c echo.Context) error {
		var err error
		output := make(map[string]string)

		params := new(handlers.CreateGroupParam)
		if err = c.Bind(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		res, err := handlers.CreateGroupUrl(params, options.GetNodeOptions(), h.Appdb)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		return c.JSON(http.StatusOK, res)
	}
}
