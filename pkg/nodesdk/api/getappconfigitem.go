package nodesdkapi

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

func (h *NodeSDKHandler) GetAppConfigItem(c echo.Context) (err error) {
	output := make(map[string]string)
	groupid := c.Param("group_id")
	if groupid == "" {
		output[ERROR_INFO] = "group_id can not be empty"
		return c.JSON(http.StatusBadRequest, output)
	}
	key := c.Param("key")
	if key == "" {
		output[ERROR_INFO] = "key can not be empty"
		return c.JSON(http.StatusBadRequest, output)
	}

	dbMgr := nodesdkctx.GetDbMgr()
	nodesdkGroupItem, err := dbMgr.GetGroupInfo(groupid)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	reqItem := new(AppConfigItem)
	reqItem.GroupId = groupid
	reqItem.Key = key
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
	getItem.ReqType = APPCONFIG_ITEM_BYKEY

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

	result := string(resultInBytes)
	return c.JSON(http.StatusOK, result)
}
