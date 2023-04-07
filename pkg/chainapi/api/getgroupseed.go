package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

// @Tags Groups
// @Summary Get group seed
// @Description get group seed from appdb
// @Produce json
// @Param group_id path string  true "Group Id"
// @Param include_chain_url query bool false "if include chain url"
// @Success 200 {object} handlers.GetGroupSeedResult
// @Router /api/v1/group/{group_id}/seed [get]
func (h *Handler) GetGroupSeedHandler(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	var params handlers.GetGroupSeedParam
	if err := cc.BindAndValidate(&params); err != nil {
		return err
	}

	seed, err := handlers.GetGroupSeed(params.GroupId, h.Appdb)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	var chainUrls []string

	if params.IncludeChainUrl {
		jwt, err := handlers.GetOrCreateGroupNodeJwt(params.GroupId)
		if err != nil {
			return rumerrors.NewInternalServerError(err)
		}

		// get chain api url
		baseUrl := cc.GetBaseURLFromRequest()
		chainapiUrl, err := utils.GetChainapiURL(baseUrl, jwt)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}
		chainUrls = append(chainUrls, chainapiUrl)
	}

	seedurl, err := handlers.GroupSeedToUrl(1, chainUrls, seed)
	if err != nil {
		return rumerrors.NewInternalServerError(fmt.Sprintf("seedurl output failed: %s", err))
	}

	result := handlers.GetGroupSeedResult{
		Seed: seedurl,
	}
	return c.JSON(http.StatusOK, result)
}
