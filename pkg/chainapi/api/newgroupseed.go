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
// @Summary NewSeed
// @Description Create a new group seed
// @Accept json
// @Produce json
// @Param data body handlers.NewGroupSeedParams true "NewGroupSeedParams"
// @Success 200 {object} handlers.NewGroupSeedResult
// @Router /api/v2/rumlite/group/newseed [post]
func (h *Handler) NewGroupSeed() echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := c.(*utils.CustomContext)

		params := new(handlers.NewGroupSeedParams)
		if err := cc.BindAndValidate(params); err != nil {
			return err
		}

		seed, err := handlers.NewGroupSeed(params, options.GetNodeOptions())
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, seed)
	}
}
