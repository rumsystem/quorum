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
		return rumerrors.NewBadRequestError(rumerrors.ErrInvalidGroupID)
	}

	nodesdkGroupItem, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(groupid)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	reqItem := new(GrpInfo)
	reqItem.GroupId = groupid

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
	getItem.ReqType = GROUP_INFO

	//just get the first one
	httpClient, err := nodesdkctx.GetCtx().GetHttpClient(nodesdkGroupItem.Group.GroupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	err = httpClient.UpdApiServer(nodesdkGroupItem.ApiUrl)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	result := new(GrpInfoNodeSDK)
	err = httpClient.RequestChainAPI(GetChainDataURI(groupid), http.MethodPost, getItem, nil, result)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	//verify groupInfo by check provider and signature

	//update db
	//save nodesdkgroupitem to db
	grpInfo, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(groupid)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}
	grpInfo.Group.HighestBlockId = result.HighestBlockId
	grpInfo.Group.HighestHeight = result.HighestHeight
	grpInfo.Group.LastUpdate = result.LatestUpdate

	if err := nodesdkctx.GetCtx().GetChainStorage().UpdGroupV2(grpInfo); err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	return c.JSON(http.StatusOK, result)
}
