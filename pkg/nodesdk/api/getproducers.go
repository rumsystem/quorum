package nodesdkapi

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

func (h *NodeSDKHandler) GetProducers(c echo.Context) (err error) {
	groupid := c.Param("group_id")
	if groupid == "" {
		return rumerrors.NewBadRequestError(rumerrors.ErrInvalidGroupID)
	}

	nodesdkGroupItem, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(groupid)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	reqItem := new(GrpProducer)
	reqItem.GroupId = groupid
	reqItem.JwtToken = JwtToken

	itemBytes, err := json.Marshal(reqItem)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	encryptData, err := getEncryptData(itemBytes, nodesdkGroupItem.Group.CipherKey)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	getItem := new(NodeSDKGetChainDataItem)
	getItem.Req = encryptData
	getItem.ReqType = GROUP_PRODUCER

	reqBytes, err := json.Marshal(getItem)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}
	//just get the first one
	httpClient, err := nodesdkctx.GetCtx().GetHttpClient(nodesdkGroupItem.Group.GroupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	err = httpClient.UpdApiServer(nodesdkGroupItem.ApiUrl)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	resultInBytes, err := httpClient.Post(GetChainDataURI(groupid), reqBytes)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	result := new([]*ProducerListItem)
	err = json.Unmarshal(resultInBytes, result)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, result)
}
