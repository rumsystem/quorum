package nodesdkapi

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

func (h *NodeSDKHandler) GetGroupInfo(c echo.Context) (err error) {
	groupid := c.Param("group_id")
	if groupid == "" {
		return rumerrors.NewBadRequestError("empty group id")
	}

	nodesdkGroupItem, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(groupid)
	if err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	reqItem := new(GrpInfo)
	reqItem.GroupId = groupid
	reqItem.JwtToken = JwtToken

	itemBytes, err := json.Marshal(reqItem)
	if err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	encryptData, err := getEncryptData(itemBytes, nodesdkGroupItem.Group.CipherKey)
	if err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	getItem := new(NodeSDKGetChainDataItem)
	getItem.Req = encryptData
	getItem.ReqType = GROUP_INFO

	reqBytes, err := json.Marshal(getItem)
	if err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	//just get the first one
	httpClient, err := nodesdkctx.GetCtx().GetHttpClient(nodesdkGroupItem.Group.GroupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	err = httpClient.UpdApiServer(nodesdkGroupItem.ApiUrl)
	if err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	resultInBytes, err := httpClient.Post(GetChainDataURI(groupid), reqBytes)
	if err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	result := new(GrpInfoNodeSDK)
	if err := json.Unmarshal(resultInBytes, result); err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	//verify groupInfo by check provider and signature

	//update db
	//save nodesdkgroupitem to db
	grpInfo, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(groupid)
	if err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}
	grpInfo.Group.HighestBlockId = result.HighestBlockId
	grpInfo.Group.HighestHeight = result.HighestHeight
	grpInfo.Group.LastUpdate = result.LatestUpdate

	if err := nodesdkctx.GetCtx().GetChainStorage().UpdGroupV2(grpInfo); err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	return c.JSON(http.StatusOK, result)
}
