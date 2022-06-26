package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	handlers "github.com/rumsystem/quorum/pkg/chainapi/handlers"
	_ "github.com/rumsystem/rumchaindata/pkg/pb" //import for swaggo
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
		cc := c.(*utils.CustomContext)
		var err error

		params := new(handlers.CreateGroupParam)
		if err := cc.BindAndValidate(params); err != nil {
			return err
		}

		res, err := handlers.CreateGroup(params, options.GetNodeOptions(), h.Appdb)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, res)
	}
}
