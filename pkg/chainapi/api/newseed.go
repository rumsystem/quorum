package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	handlers "github.com/rumsystem/quorum/pkg/chainapi/handlers"
	_ "github.com/rumsystem/quorum/pkg/pb" //import for swaggo
)

// @Tags Groups
// @Summary CreateGroupUrl
// @Description Create a new group
// @Accept json
// @Produce json
// @Param data body handlers.CreateGroupParam true "GroupInfo"
// @Success 200 {object} handlers.CreateGroupResult
// @Router /api/v1/group [post]
func (h *Handler) NewSeed() echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := c.(*utils.CustomContext)

		params := new(handlers.NewSeedParams)
		if err := cc.BindAndValidate(params); err != nil {
			return err
		}

		seed, err := handlers.NewSeed(params, options.GetNodeOptions())
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, seed)
	}
}
