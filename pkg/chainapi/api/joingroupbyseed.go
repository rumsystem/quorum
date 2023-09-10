package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	handlers "github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags Groups
// @Summary JoinGroupBySeed
// @Description Join a group by using group seed
// @Accept json
// @Produce json
// @Param data body handlers.JoinGroupBySeedParam true "JoinGroupBySeedParam"
// @Success 200 {object} JoinGroupBySeedResult
// @Router /api/v2/group/join [post]
func (h *Handler) JoinGroupBySeed() echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := c.(*utils.CustomContext)

		params := new(handlers.JoinGroupBySeedParams)
		if err := cc.BindAndValidate(params); err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		result, err := handlers.JoinGroupBySeed(params, options.GetNodeOptions())
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, result)
	}
}
