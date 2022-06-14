package nodesdkapi

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

const GET_BLOCK_URI string = "/api/v1/block"

func (h *NodeSDKHandler) GetBlock() echo.HandlerFunc {
	return func(c echo.Context) error {
		var err error
		output := make(map[string]string)

		groupid := c.Param("group_id")
		if groupid == "" {
			output[ERROR_INFO] = "group_id can't be nil."
			return c.JSON(http.StatusBadRequest, output)
		}

		blockid := c.Param("block_id")
		if blockid == "" {
			output[ERROR_INFO] = "block_id can't be nil."
			return c.JSON(http.StatusBadRequest, output)
		}

		nodesdkGroupItem, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(groupid)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		//just get the first one
		httpClient, err := nodesdkctx.GetCtx().GetHttpClient(nodesdkGroupItem.Group.GroupId)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		err = httpClient.UpdApiServer(nodesdkGroupItem.ApiUrl)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		uri := GET_BLOCK_URI + "/" + groupid + "/" + blockid

		resultInBytes, err := httpClient.Get(uri)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		block := new(quorumpb.Block)
		err = json.Unmarshal(resultInBytes, block)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		return c.JSON(http.StatusOK, block)
	}
}
