package nodesdkapi

import (
	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type GetGroupSeedParams struct {
	GroupId string `json:"group_id" validate:"required"`
}

func (h *NodeSDKHandler) GetGroupSeed() echo.HandlerFunc {
	return func(c echo.Context) error {
		groupid := c.Param("group_id")
		if groupid == "" {
			return rumerrors.NewBadRequestError(rumerrors.ErrInvalidGroupID)
		}

		pbseed, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupSeed(groupid)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}
		seed := handlers.FromPbGroupSeed(pbseed)

		seedurl, err := handlers.GroupSeedToUrl(1, []string{}, &seed)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}
		output["seed"] = seedurl

		return c.JSON(http.StatusOK, output)
	}
}
