package api

import (
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

const GROUP_INFO string = "group_info"

const AUTH_TYPE string = "auth_type"
const AUTH_ALLOWLIST string = "auth_allowlist"
const AUTH_DENYLIST string = "auth_denylist"

const APPCONFIG_KEYLIST string = "appconfig_listlist"
const APPCONFIG_ITEM_BYKEY string = "appconfig_item_bykey"

const ANNOUNCED_PRODUCER string = "announced_producer"
const ANNOUNCED_USER string = "announced_user"
const GROUP_PRODUCER string = "group_producer"

type GetDataNodeSDKItem struct {
	GroupId string `param:"group_id" validate:"required"`
	ReqType string
	Req     []byte
}

type GrpInfo struct {
	GroupId  string
	JwtToken string
}

type AuthTypeItem struct {
	GroupId  string
	TrxType  string
	JwtToken string
}

type AuthAllowListItem struct {
	GroupId  string
	JwtToken string
}

type AuthDenyListItem struct {
	GroupId  string
	JwtToken string
}

type AppConfigKeyListItem struct {
	GroupId  string
	JwtToken string
}

type AppConfigItem struct {
	GroupId  string
	Key      string
	JwtToken string
}

type AnnGrpProducer struct {
	GroupId  string
	JwtToken string
}

type GrpProducer struct {
	GroupId  string
	JwtToken string
}

type AnnGrpUser struct {
	GroupId    string
	SignPubkey string
	JwtToken   string
}

type GrpInfoNodeSDK struct {
	GroupId        string
	Owner          string
	HighestBlockId string
	HighestHeight  int64
	LatestUpdate   int64
	Provider       string
	Singature      string
}

func (h *Handler) GetDataNSdk(c echo.Context) (err error) {
	if is_user_blocked(c) {
		return c.JSON(http.StatusForbidden, "")
	}

	output := make(map[string]string)
	getDataNodeSDKItem := new(GetDataNodeSDKItem)

	if err = c.Bind(getDataNodeSDKItem); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[getDataNodeSDKItem.GroupId]; ok {
		if group.Item.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
			output[ERROR_INFO] = "FUNCTION_NOT_SUPPORTED"
			return c.JSON(http.StatusBadRequest, output)
		}

		ciperKey, err := hex.DecodeString(group.Item.CipherKey)
		if err != nil {
			output[ERROR_INFO] = "CHAINSDK_INTERNAL_ERROR"
			return c.JSON(http.StatusBadRequest, output)
		}

		decryptData, err := localcrypto.AesDecode(getDataNodeSDKItem.Req, ciperKey)
		if err != nil {
			output[ERROR_INFO] = "DECRYPT_DATA_FAILED"
			return c.JSON(http.StatusBadRequest, output)
		}

		switch getDataNodeSDKItem.ReqType {
		case AUTH_TYPE:
			item := new(AuthTypeItem)
			err = json.Unmarshal(decryptData, item)
			if err != nil {
				output[ERROR_INFO] = "INVALID_DATA"
				return c.JSON(http.StatusBadRequest, output)
			}
			if item.JwtToken != NodeSDKJwtToken {
				output[ERROR_INFO] = "INVALID_JWT_TOKEN"
				return c.JSON(http.StatusBadRequest, output)
			}
			res, err := handlers.GetChainTrxAuthMode(h.ChainAPIdb, item.GroupId, item.TrxType)
			if err != nil {
				output[ERROR_INFO] = err.Error()
				return c.JSON(http.StatusBadRequest, output)
			}
			return c.JSON(http.StatusOK, res)
		case AUTH_ALLOWLIST:
			item := new(AuthAllowListItem)
			err = json.Unmarshal(decryptData, item)
			if err != nil {
				output[ERROR_INFO] = "INVALID_DATA"
				return c.JSON(http.StatusBadRequest, output)
			}
			if item.JwtToken != NodeSDKJwtToken {
				output[ERROR_INFO] = "INVALID_JWT_TOKEN"
				return c.JSON(http.StatusBadRequest, output)
			}
			res, err := handlers.GetChainTrxAllowList(h.ChainAPIdb, item.GroupId)
			if err != nil {
				output[ERROR_INFO] = err.Error()
				return c.JSON(http.StatusBadRequest, output)
			}
			return c.JSON(http.StatusOK, res)
		case AUTH_DENYLIST:
			item := new(AuthDenyListItem)
			err = json.Unmarshal(decryptData, item)
			if err != nil {
				output[ERROR_INFO] = "INVALID_DATA"
				return c.JSON(http.StatusBadRequest, output)
			}
			if item.JwtToken != NodeSDKJwtToken {
				output[ERROR_INFO] = "INVALID_JWT_TOKEN"
				return c.JSON(http.StatusBadRequest, output)
			}
			res, err := handlers.GetChainTrxDenyList(h.ChainAPIdb, item.GroupId)
			if err != nil {
				output[ERROR_INFO] = err.Error()
				return c.JSON(http.StatusBadRequest, output)
			}
			return c.JSON(http.StatusOK, res)
		case APPCONFIG_KEYLIST:
			item := new(AppConfigKeyListItem)
			err = json.Unmarshal(decryptData, item)
			if err != nil {
				output[ERROR_INFO] = "INVALID_DATA"
				return c.JSON(http.StatusBadRequest, output)
			}
			if item.JwtToken != NodeSDKJwtToken {
				output[ERROR_INFO] = "INVALID_JWT_TOKEN"
				return c.JSON(http.StatusBadRequest, output)
			}
			res, err := handlers.GetAppConfigKeyList(item.GroupId)
			if err != nil {
				output[ERROR_INFO] = err.Error()
				return c.JSON(http.StatusBadRequest, output)
			}
			return c.JSON(http.StatusOK, res)
		case APPCONFIG_ITEM_BYKEY:
			item := new(AppConfigItem)
			err = json.Unmarshal(decryptData, item)
			if err != nil {
				output[ERROR_INFO] = "INVALID_DATA"
				return c.JSON(http.StatusBadRequest, output)
			}
			if item.JwtToken != NodeSDKJwtToken {
				output[ERROR_INFO] = "INVALID_JWT_TOKEN"
				return c.JSON(http.StatusBadRequest, output)
			}
			res, err := handlers.GetAppConfigKey(item.Key, item.GroupId)
			if err != nil {
				output[ERROR_INFO] = err.Error()
				return c.JSON(http.StatusBadRequest, output)
			}
			return c.JSON(http.StatusOK, res)
		case ANNOUNCED_PRODUCER:
			item := new(AnnGrpProducer)
			err = json.Unmarshal(decryptData, item)
			if err != nil {
				output[ERROR_INFO] = "INVALID_DATA"
				return c.JSON(http.StatusBadRequest, output)
			}
			if item.JwtToken != NodeSDKJwtToken {
				output[ERROR_INFO] = "INVALID_JWT_TOKEN"
				return c.JSON(http.StatusBadRequest, output)
			}
			res, err := handlers.GetAnnouncedGroupProducer(h.ChainAPIdb, item.GroupId)
			if err != nil {
				output[ERROR_INFO] = err.Error()
				return c.JSON(http.StatusBadRequest, output)
			}
			return c.JSON(http.StatusOK, res)
		case ANNOUNCED_USER:
			item := new(AnnGrpUser)
			err = json.Unmarshal(decryptData, item)
			if err != nil {
				output[ERROR_INFO] = "INVALID_DATA"
				return c.JSON(http.StatusBadRequest, output)
			}
			if item.JwtToken != NodeSDKJwtToken {
				output[ERROR_INFO] = "INVALID_JWT_TOKEN"
				return c.JSON(http.StatusBadRequest, output)
			}
			if item.SignPubkey == "" {
				res, err := handlers.GetAnnouncedGroupUsers(h.ChainAPIdb, item.GroupId)
				if err != nil {
					output[ERROR_INFO] = err.Error()
					return c.JSON(http.StatusBadRequest, output)
				}
				return c.JSON(http.StatusOK, res)
			} else {
				res, err := handlers.GetAnnouncedGroupUser(item.GroupId, item.SignPubkey)
				if err != nil {
					output[ERROR_INFO] = err.Error()
					return c.JSON(http.StatusBadRequest, output)
				}
				return c.JSON(http.StatusOK, res)
			}
		case GROUP_PRODUCER:
			item := new(GrpProducer)
			err = json.Unmarshal(decryptData, item)
			if err != nil {
				output[ERROR_INFO] = "INVALID_DATA"
				return c.JSON(http.StatusBadRequest, output)
			}
			if item.JwtToken != NodeSDKJwtToken {
				output[ERROR_INFO] = "INVALID_JWT_TOKEN"
				return c.JSON(http.StatusBadRequest, output)
			}
			res, err := handlers.GetGroupProducers(h.ChainAPIdb, item.GroupId)
			if err != nil {
				output[ERROR_INFO] = err.Error()
				return c.JSON(http.StatusBadRequest, output)
			}
			return c.JSON(http.StatusOK, res)
		case GROUP_INFO:
			item := new(GrpInfo)
			err = json.Unmarshal(decryptData, item)
			if err != nil {
				output[ERROR_INFO] = "INVALID_DATA"
				return c.JSON(http.StatusBadRequest, output)
			}
			if item.JwtToken != NodeSDKJwtToken {
				output[ERROR_INFO] = "INVALID_JWT_TOKEN"
				return c.JSON(http.StatusBadRequest, output)
			}

			if grp, ok := groupmgr.Groups[item.GroupId]; ok {
				grpInfo := new(GrpInfoNodeSDK)
				grpInfo.GroupId = grp.Item.GroupId
				grpInfo.Owner = grp.Item.OwnerPubKey
				grpInfo.Provider = grp.Item.UserSignPubkey
				grpInfo.LatestUpdate = grp.Item.LastUpdate
				grpInfo.HighestBlockId = grp.Item.HighestBlockId
				grpInfo.HighestHeight = grp.Item.HighestHeight

				/*
					//Did we really need a sign from fullnode ?
					Sign hash with fullnode pubkey
					groInfoBytes, err := json.Marshal(grpInfo)
					if err != nil {
						output[ERROR_INFO] = "INTERNAL_ERROR"
						return c.JSON(http.StatusBadRequest, output)
					}
					hash := localcrypto.Hash(groInfoBytes)
					grpInfo.Singature = "FAKE_SIGN"
				*/

				return c.JSON(http.StatusOK, grpInfo)
			} else {
				output[ERROR_INFO] = "INVALID_GROUP"
				return c.JSON(http.StatusBadRequest, output)
			}
		default:
			output[ERROR_INFO] = "UNKNOWN_REQ_TYPE"
			return c.JSON(http.StatusBadRequest, output)
		}
	} else {
		output[ERROR_INFO] = "INVALID_GROUP"
		return c.JSON(http.StatusBadRequest, output)
	}
}
