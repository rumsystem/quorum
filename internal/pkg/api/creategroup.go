package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	"github.com/rumsystem/quorum/internal/pkg/options"
)

// @Tags Groups
// @Summary CreateGroup
// @Description Create a new group
// @Accept json
// @Produce json
// @Param data body handlers.CreateGroupParam true "GroupInfo"
// @Success 200 {object} handlers.GroupSeed
// @Router /api/v1/group [post]
func (h *Handler) CreateGroup() echo.HandlerFunc {
	return func(c echo.Context) error {
		var err error
		output := make(map[string]string)

		params := new(handlers.CreateGroupParam)
		if err = c.Bind(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		res, err := handlers.CreateGroup(params, options.GetNodeOptions(), h.Appdb)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		return c.JSON(http.StatusOK, res)
	}
}
