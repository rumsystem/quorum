package api

import (
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
)

const GROUP_INFO string = "group_info"

const AUTH_TYPE string = "auth_type"
const AUTH_ALLOWLIST string = "auth_allowlist"
const AUTH_DENYLIST string = "auth_denylist"

const APPCONFIG_KEYLIST string = "appconfig_listlist"
const APPCONFIG_ITEM_BYKEY string = "appconfig_item_bykey"

const GROUP_PRODUCER string = "group_producer"

type GetDataNodeSDKItem struct {
	GroupId string `param:"group_id" json:"-" validate:"required"`
	ReqType string `json:"ReqType" validate:"required,oneof=auth_type auth_allowlist auth_denylist appconfig_listlist appconfig_item_bykey announced_producer announced_user group_producer group_info"`
	Req     []byte `json:"Req" validate:"required" swaggertype:"primitive,string"` // base64 encoded req
}

type GrpInfo struct {
	GroupId string
}

type AuthTypeItem struct {
	GroupId string
	TrxType string
}

type AuthAllowListItem struct {
	GroupId string
}

type AuthDenyListItem struct {
	GroupId string
}

type AppConfigKeyListItem struct {
	GroupId string
}

type AppConfigItem struct {
	GroupId string
	Key     string
}

type AnnGrpProducer struct {
	GroupId string
}

type GrpProducer struct {
	GroupId string
}

type AnnGrpUser struct {
	GroupId    string
	SignPubkey string
}

type GrpInfoNodeSDK struct {
	GroupId      string
	Owner        string
	Epoch        int64
	LatestUpdate int64
	Provider     string
	Singature    string
}

// @Tags NodeAPI
// @Summary GetDataNSdk
// @Description get chain data
// @Accept  json
// @Produce json
// @Param   group_id path string true "Group Id"
// @Param   get_data_params  body GetDataNodeSDKItem true  "get chain data params"
// @Success 200 {object} interface{}
// @Router  /api/v1/node/getchaindata/{group_id} [post]
func (h *Handler) GetDataNSdk(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)

	getDataNodeSDKItem := new(GetDataNodeSDKItem)
	if err := cc.BindAndValidate(getDataNodeSDKItem); err != nil {
		return rumerrors.NewBadRequestError(err)
	}
	c.Logger().Debug("GetDataNSdk request payload: %+v", *getDataNodeSDKItem)

	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[getDataNodeSDKItem.GroupId]
	if !ok {
		return rumerrors.NewBadRequestError("INVALID_GROUP")
	}

	ciperKey, err := hex.DecodeString(group.Item.CipherKey)
	if err != nil {
		return rumerrors.NewBadRequestError("CHAINSDK_INTERNAL_ERROR")
	}

	decryptData, err := localcrypto.AesDecode(getDataNodeSDKItem.Req, ciperKey)
	if err != nil {
		return rumerrors.NewBadRequestError("DECRYPT_DATA_FAILED")
	}

	switch getDataNodeSDKItem.ReqType {
	case AUTH_TYPE:
		item := new(AuthTypeItem)
		err = json.Unmarshal(decryptData, item)
		if err != nil {
			return rumerrors.NewBadRequestError("INVALID_DATA")
		}
		res, err := handlers.GetChainTrxAuthMode(h.ChainAPIdb, item.GroupId, item.TrxType)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}
		return c.JSON(http.StatusOK, res)
	case AUTH_ALLOWLIST:
		item := new(AuthAllowListItem)
		err = json.Unmarshal(decryptData, item)
		if err != nil {
			return rumerrors.NewBadRequestError("INVALID_DATA")
		}
		res, err := handlers.GetChainTrxAllowList(h.ChainAPIdb, item.GroupId)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}
		return c.JSON(http.StatusOK, res)
	case AUTH_DENYLIST:
		item := new(AuthDenyListItem)
		err = json.Unmarshal(decryptData, item)
		if err != nil {
			return rumerrors.NewBadRequestError("INVALID_DATA")
		}
		res, err := handlers.GetChainTrxDenyList(h.ChainAPIdb, item.GroupId)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}
		return c.JSON(http.StatusOK, res)
	case APPCONFIG_KEYLIST:
		item := new(AppConfigKeyListItem)
		err = json.Unmarshal(decryptData, item)
		if err != nil {
			return rumerrors.NewBadRequestError("INVALID_DATA")
		}
		res, err := handlers.GetAppConfigKeyList(item.GroupId)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}
		return c.JSON(http.StatusOK, res)
	case APPCONFIG_ITEM_BYKEY:
		item := new(AppConfigItem)
		err = json.Unmarshal(decryptData, item)
		if err != nil {
			return rumerrors.NewBadRequestError("INVALID_DATA")
		}
		res, err := handlers.GetAppConfigByKey(item.Key, item.GroupId)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}
		return c.JSON(http.StatusOK, res)
	case GROUP_PRODUCER:
		item := new(GrpProducer)
		err = json.Unmarshal(decryptData, item)
		if err != nil {
			return rumerrors.NewBadRequestError("INVALID_DATA")
		}
		res, err := handlers.GetGroupProducers(h.ChainAPIdb, item.GroupId)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}
		return c.JSON(http.StatusOK, res)
	case GROUP_INFO:
		item := new(GrpInfo)
		err = json.Unmarshal(decryptData, item)
		if err != nil {
			return rumerrors.NewBadRequestError("INVALID_DATA")
		}

		if grp, ok := groupmgr.Groups[item.GroupId]; ok {
			grpInfo := new(GrpInfoNodeSDK)
			grpInfo.GroupId = grp.Item.GroupId
			grpInfo.Owner = grp.Item.OwnerPubKey
			grpInfo.Provider = grp.Item.UserSignPubkey
			grpInfo.LatestUpdate = grp.Item.LastUpdate
			//grpInfo.Epoch = grp.Item.Epoch

			return c.JSON(http.StatusOK, grpInfo)
		} else {
			return rumerrors.NewBadRequestError("INVALID_GROUP")
		}
	default:
		return rumerrors.NewBadRequestError("UNKNOWN_REQ_TYPE")
	}
}
