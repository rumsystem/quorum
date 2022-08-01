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
// @Summary CreateGroupUrl
// @Description Create a new group
// @Accept json
// @Produce json
// @Param data body handlers.CreateGroupParam true "GroupInfo"
// @Success 200 {object} handlers.CreateGroupResult
// @Router /api/v1/group [post]
func (h *Handler) CreateGroupUrl() echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := c.(*utils.CustomContext)

		params := new(handlers.CreateGroupParam)
		if err := cc.BindAndValidate(params); err != nil {
			return err
		}

		baseUrl := cc.GetBaseURLFromRequest()
		res, err := handlers.CreateGroupUrl(baseUrl, params, options.GetNodeOptions(), h.Appdb)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, res)
	}
}
