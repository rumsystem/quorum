package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	handlers "github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags Management
// @Summary UpdConsensus
// @Description	Update group consensus configuration
// @Accept json
// @Produce json
// @Param data body handlers.ReqConsensusChangeParam true "UpdConsensusParam"
// @Success 200 {object} handlers.ReqConsensusChangeResult
// @Router /api/v1/group/updconsensus [post]
func (h *Handler) ReqConsensusChange(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(handlers.ReqConsensusChangeParam)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	res, err := handlers.ReqConsensusChange(h.ChainAPIdb, params)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)

}
