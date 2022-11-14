package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
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
	cc := c.(*utils.CustomContext)
	params := new(handlers.GrpProducerParam)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	var sudo bool
	if c.QueryParams().Get("sudo") == "" {
		sudo = false
	} else {
		v, err := strconv.ParseBool(c.Param("sudo"))
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}
		sudo = v
	}

	res, err := handlers.GroupProducer(h.ChainAPIdb, params, sudo)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, res)
}
