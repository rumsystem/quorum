package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type ChainConfigParams struct {
	GroupId string `from:"group_id" json:"group_id"  validate:"required,uuid4" example:"ac0eea7c-2f3c-4c67-80b3-136e46b924a8"`
	Type    string `from:"type"     json:"type"      validate:"required,oneof=set_trx_auth_mode upd_alw_list upd_dny_list" example:"upd_alw_list"`
	Config  string `from:"config"   json:"config"    validate:"required" example:"{\"action\":\"add\",  \"pubkey\":\"CAISIQNGAO67UTFSuWzySHKdy4IjBI/Q5XDMELPUSxHpBwQDcQ==\", \"trx_type\":[\"post\", \"announce\", \"req_block_forward\", \"req_block_backward\", \"ask_peerid\"]}"`
	Memo    string `from:"memo"     json:"memo" example:"comment/remark"`
}

type TrxAuthModeParams struct {
	TrxType     string `from:"trx_type"      json:"trx_type"     validate:"required,oneof=POST ANNOUNCE PRODUCER REQ_BLOCK USER CHAIN_CONFIG APP_CONFIG" example:"POST"`
	TrxAuthMode string `from:"trx_auth_mode" json:"trx_auth_mode" validate:"required,oneof=follow_alw_list follow_dny_list" example:"follow_alw_list"`
}
type ChainSendTrxRuleListItemParams struct {
	Action  string   `from:"action"   json:"action"   validate:"required,oneof=add remove" example:"add"`
	Pubkey  string   `from:"pubkey"   json:"pubkey"   validate:"required" example:"CAISIQNGAO67UTFSuWzySHKdy4IjBI/Q5XDMELPUSxHpBwQDcQ=="`
	TrxType []string `from:"trx_type" json:"trx_type" validate:"required"` // Example: ["POST", "ANNOUNCE"]
}

type ChainConfigResult struct {
	GroupId          string `json:"group_id"     validate:"required,uuid4" example:"b3e1800a-af6e-4c67-af89-4ddcf831b6f7"`
	GroupOwnerPubkey string `json:"owner_pubkey" validate:"required" example:"CAISIQPLW/J9xgdMWoJxFttChoGOOld8TpChnGFFyPADGL+0JA=="`
	Sign             string `json:"signature"    validate:"required" example:"30440220089276796ceeef3a2c413bd89249475c2ecd8be4f2cb0ee3d19903fc45a7386b02206561bfdfb0338a9d022619dd8064e9a3496c1ea768f344e3c3850f8a907cdc73"`
	TrxId            string `json:"trx_id"       validate:"required" example:"90e9818a-2e23-4248-93e3-d4ba1b100f4f"`
}

func MgrChainConfig(params *ChainConfigParams, sudo bool) (*ChainConfigResult, error) {
	validate := validator.New()
	if err := validate.Struct(params); err != nil {
		return nil, err
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[params.GroupId]; !ok {
		return nil, rumerrors.ErrGroupNotFound
	} else if group.Item.OwnerPubKey != group.Item.UserSignPubkey {
		return nil, rumerrors.ErrOnlyGroupOwner
	}

	group := groupmgr.Groups[params.GroupId]

	if group.Item.UserSignPubkey != group.Item.OwnerPubKey {
		return nil, errors.New("Only group owner can run sudo mode")
	}

	ks := nodectx.GetNodeCtx().Keystore
	base64key, err := ks.GetEncodedPubkey(params.GroupId, localcrypto.Sign)
	groupSignPubkey, err := base64.RawURLEncoding.DecodeString(base64key)
	//pubkeybytes, err := hex.DecodeString(hexkey)
	//p2ppubkey, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
	//groupSignPubkey, err := p2pcrypto.MarshalPublicKey(p2ppubkey)
	if err != nil {
		return nil, errors.New("group key can't be decoded, err:" + err.Error())
	}

	var configItem quorumpb.ChainConfigItem
	configItem = quorumpb.ChainConfigItem{}
	configItem.GroupId = params.GroupId

	if params.Type == strings.ToLower(quorumpb.ChainConfigType_SET_TRX_AUTH_MODE.String()) {
		dataParams := TrxAuthModeParams{}
		err := json.Unmarshal([]byte(params.Config), &dataParams)
		if err != nil {
			return nil, err
		}

		if err := validate.Struct(dataParams); err != nil {
			return nil, err
		}

		dataItem := quorumpb.SetTrxAuthModeItem{}
		dataItem.Type, err = getTrxTypeByString(dataParams.TrxType)
		if err != nil {
			return nil, err
		}

		if dataParams.TrxAuthMode == strings.ToLower(quorumpb.TrxAuthMode_FOLLOW_ALW_LIST.String()) {
			dataItem.Mode = quorumpb.TrxAuthMode_FOLLOW_ALW_LIST
		} else if dataParams.TrxAuthMode == strings.ToLower(quorumpb.TrxAuthMode_FOLLOW_DNY_LIST.String()) {
			dataItem.Mode = quorumpb.TrxAuthMode_FOLLOW_DNY_LIST
		} else {
			return nil, errors.New("Unsupported trx_auth_mode")
		}

		encodedcontent, err := proto.Marshal(&dataItem)
		if err != nil {
			return nil, err
		}

		configItem.Type = quorumpb.ChainConfigType_SET_TRX_AUTH_MODE
		configItem.Data = encodedcontent
	} else if params.Type == strings.ToLower(quorumpb.ChainConfigType_UPD_ALW_LIST.String()) ||
		params.Type == strings.ToLower(quorumpb.ChainConfigType_UPD_DNY_LIST.String()) {
		dataParams := ChainSendTrxRuleListItemParams{}
		err := json.Unmarshal([]byte(params.Config), &dataParams)
		if err != nil {
			return nil, err
		}

		if err := validate.Struct(dataParams); err != nil {
			return nil, err
		}
		dataItem := quorumpb.ChainSendTrxRuleListItem{}
		if dataParams.Action == "add" {
			dataItem.Action = quorumpb.ActionType_ADD
		} else {
			dataItem.Action = quorumpb.ActionType_REMOVE
		}
		dataItem.Pubkey = dataParams.Pubkey
		var trxTypes []quorumpb.TrxType
		for _, typ := range dataParams.TrxType {
			trxType, err := getTrxTypeByString(typ)
			if err != nil {
				return nil, err
			}
			trxTypes = append(trxTypes, trxType)
		}
		dataItem.Type = trxTypes
		encodedcontent, err := proto.Marshal(&dataItem)
		if err != nil {
			return nil, err
		}
		if params.Type == strings.ToLower(quorumpb.ChainConfigType_UPD_ALW_LIST.String()) {
			configItem.Type = quorumpb.ChainConfigType_UPD_ALW_LIST
		} else {
			configItem.Type = quorumpb.ChainConfigType_UPD_DNY_LIST
		}
		configItem.Data = encodedcontent
	} else {
		return nil, errors.New("Type not supported")
	}

	var buffer bytes.Buffer
	buffer.Write([]byte(configItem.GroupId))
	buffer.Write([]byte(configItem.Type.String()))
	buffer.Write(configItem.Data)
	buffer.Write(groupSignPubkey)
	bt := make([]byte, 8)
	binary.LittleEndian.PutUint64(bt, uint64(configItem.TimeStamp))
	buffer.Write(bt)
	buffer.Write([]byte(configItem.Memo))

	hash := localcrypto.Hash(buffer.Bytes())

	signature, err := ks.EthSignByKeyName(params.GroupId, hash)
	if err != nil {
		return nil, err
	}

	configItem.Memo = params.Memo
	configItem.TimeStamp = time.Now().UnixNano()
	configItem.OwnerPubkey = group.Item.OwnerPubKey
	configItem.OwnerSignature = hex.EncodeToString(signature)
	trxId, err := group.UpdChainConfig(&configItem)
	if err != nil {
		return nil, err
	}

	result := &ChainConfigResult{GroupId: configItem.GroupId, GroupOwnerPubkey: base64key, Sign: hex.EncodeToString(signature), TrxId: trxId}
	return result, nil

}

func getTrxTypeByString(typ string) (quorumpb.TrxType, error) {
	switch strings.ToUpper(typ) {
	case "POST":
		return quorumpb.TrxType_POST, nil
	case "ANNOUNCE":
		return quorumpb.TrxType_ANNOUNCE, nil
	case "PRODUCER":
		return -1, errors.New("this trx type can not be configured")
	case "REQ_BLOCK":
		return quorumpb.TrxType_REQ_BLOCK, nil
	case "USER":
		return -1, errors.New("this trx type can not be configured")
	case "CHAIN_CONFIG":
		return -1, errors.New("this trx type can not be configured")
	case "APP_CONFIG":
		return -1, errors.New("this trx type can not be configured")
	default:
		return -1, errors.New("Unsupported TrxType")
	}
}
