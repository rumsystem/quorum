package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	handlers "github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags User
// @Summary Request psync with other producers
// @Description Request a round of psync amont other producers
// @Accept None
// @Produce json
// @Param data body None
// @Success 200 {object} handlers.ReqPSync
// @Router /api/v1/group/reqpsync [post]
func (h *Handler) ReqPSync(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(handlers.ReqPSyncParam)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	res, err := handlers.ReqPSyncHandler(params)

	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}
