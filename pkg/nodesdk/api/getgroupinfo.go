package nodesdkapi

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

func (h *NodeSDKHandler) GetGroupInfo(c echo.Context) (err error) {
	output := make(map[string]string)
	groupid := c.Param("group_id")

	if groupid == "" {
		output[ERROR_INFO] = "group_id can not be empty"
		return c.JSON(http.StatusBadRequest, output)
	}

	dbMgr := nodesdkctx.GetDbMgr()
	nodesdkGroupItem, err := dbMgr.GetGroupInfo(groupid)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	reqItem := new(GrpInfo)
	reqItem.GroupId = groupid
	reqItem.JwtToken = JwtToken

	itemBytes, err := json.Marshal(reqItem)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	encryptData, err := getEncryptData(itemBytes, nodesdkGroupItem.Group.CipherKey)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	getItem := new(NodeSDKGetChainDataItem)
	getItem.GroupId = groupid
	getItem.Req = encryptData
	getItem.ReqType = GROUP_INFO

	reqBytes, err := json.Marshal(getItem)
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

	resultInBytes, err := httpClient.Post(GET_CHAIN_DATA_URI, reqBytes)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	result := new(GrpInfoNodeSDK)
	err = json.Unmarshal(resultInBytes, result)

	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	//verify groupInfo by check provider and signature

	//update db
	//save nodesdkgroupitem to db
	grpInfo, err := dbMgr.GetGroupInfo(groupid)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}
	grpInfo.Group.HighestBlockId = result.HighestBlockId
	grpInfo.Group.HighestHeight = result.HighestHeight
	grpInfo.Group.LastUpdate = result.LatestUpdate

	err = dbMgr.UpdGroup(grpInfo)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, result)
}
