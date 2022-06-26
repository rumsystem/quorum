package nodesdkapi

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

func (h *NodeSDKHandler) GetAnnouncedUsers(c echo.Context) (err error) {
	groupid := c.Param("group_id")

	if groupid == "" {
		return rumerrors.NewBadRequestError(rumerrors.ErrEmptyGroupID.Error())
	}

	signPubkey := c.Param("sign_pubkey")

	nodesdkGroupItem, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(groupid)
	if err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	reqItem := new(AnnGrpUser)
	reqItem.GroupId = groupid
	reqItem.SignPubkey = signPubkey
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
	getItem.ReqType = ANNOUNCED_USER

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

	result := new([]*AnnGrpUser)
	err = json.Unmarshal(resultInBytes, result)
	if err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	return c.JSON(http.StatusOK, result)
}
