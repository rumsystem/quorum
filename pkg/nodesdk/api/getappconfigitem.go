package nodesdkapi

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type GetAppConfigResultItem struct {
	Name        string
	Type        string
	Value       string
	OwnerPubkey string
	Memo        string
	Timestamp   int
}

func (h *NodeSDKHandler) GetAppConfigItem(c echo.Context) (err error) {
	groupid := c.Param("group_id")
	if groupid == "" {
		return rumerrors.NewBadRequestError(rumerrors.ErrEmptyGroupID.Error())
	}

	key := c.Param("key")
	if key == "" {
		return rumerrors.NewBadRequestError("empty key")
	}

	nodesdkGroupItem, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(groupid)
	if err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	reqItem := new(AppConfigItem)
	reqItem.GroupId = groupid
	reqItem.Key = key
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
	getItem.ReqType = APPCONFIG_ITEM_BYKEY

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

	result := new(*GetAppConfigResultItem)
	if err := json.Unmarshal(resultInBytes, result); err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	return c.JSON(http.StatusOK, result)
}
