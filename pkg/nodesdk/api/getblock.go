package nodesdkapi

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

const GET_BLOCK_URI string = "/api/v1/block"

func (h *NodeSDKHandler) GetBlock() echo.HandlerFunc {
	return func(c echo.Context) error {
		groupid := c.Param("group_id")
		if groupid == "" {
			return rumerrors.NewBadRequestError(rumerrors.ErrInvalidGroupID)
		}

		blockid := c.Param("block_id")
		if blockid == "" {
			return rumerrors.NewBadRequestError(rumerrors.ErrEmptyBlockID)
		}

		nodesdkGroupItem, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(groupid)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		//just get the first one
		httpClient, err := nodesdkctx.GetCtx().GetHttpClient(nodesdkGroupItem.Group.GroupId)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		if err := httpClient.UpdApiServer(nodesdkGroupItem.ApiUrl); err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		uri := GET_BLOCK_URI + "/" + groupid + "/" + blockid

		resultInBytes, err := httpClient.Get(uri)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		block := new(quorumpb.Block)
		if err := json.Unmarshal(resultInBytes, block); err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		return c.JSON(http.StatusOK, block)
	}
}
