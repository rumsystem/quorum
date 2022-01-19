package handlers

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type ChainConfigParams struct {
	GroupId string `from:"group_id" json:"group_id"  validate:"required"`
	Type    string `from:"type"     json:"type"      validate:"required"`
	Config  string `from:"config"   json:"config"    validate:"required"`
	Memo    string `from:"memo"     json:"memo"`
}

type TrxAuthModeParams struct {
	TrxType     string `from:"trx_type"      json:"trx_type"     validate:"required"`
	TrxAuthMode string `from:"trx_auth_mode" json:"trx_auth_mode" validate:"required"`
}
type ChainSendTrxRuleListItemParams struct {
	Action  string   `from:"action"   json:"action"   validate:"required,oneof=add remove"`
	Pubkey  string   `from:"pubkey"   json:"pubkey"   validate:"required"`
	TrxType []string `from:"trx_type" json:"trx_type" validate:"required"`
}

type ChainConfigResult struct {
	GroupId          string `json:"group_id"     validate:"required"`
	GroupOwnerPubkey string `json:"owner_pubkey" validate:"required"`
	Sign             string `json:"signature"    validate:"required"`
	TrxId            string `json:"trx_id"       validate:"required"`
}

func MgrChainConfig(params *ChainConfigParams) (*ChainConfigResult, error) {
	validate := validator.New()
	if err := validate.Struct(params); err != nil {
		return nil, err
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[params.GroupId]; !ok {
		return nil, errors.New("Can not find group")
	} else if group.Item.OwnerPubKey != group.Item.UserSignPubkey {
		return nil, errors.New("Only group owner can change chain configuration")
	}

	group := groupmgr.Groups[params.GroupId]

	ks := nodectx.GetNodeCtx().Keystore
	hexkey, err := ks.GetEncodedPubkey(params.GroupId, localcrypto.Sign)
	pubkeybytes, err := hex.DecodeString(hexkey)
	p2ppubkey, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
	groupSignPubkey, err := p2pcrypto.MarshalPublicKey(p2ppubkey)
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

	hash := chain.Hash(buffer.Bytes())

	signature, err := ks.SignByKeyName(params.GroupId, hash)

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

	result := &ChainConfigResult{GroupId: configItem.GroupId, GroupOwnerPubkey: p2pcrypto.ConfigEncodeKey(groupSignPubkey), Sign: hex.EncodeToString(signature), TrxId: trxId}
	return result, nil

}

func getTrxTypeByString(typ string) (quorumpb.TrxType, error) {
	switch strings.ToUpper(typ) {
	case "POST":
		return quorumpb.TrxType_POST, nil
	case "SCHEMA":
		return -1, errors.New("this trx type can not be configured")
	case "PRODUCER":
		return -1, errors.New("this trx type can not be configured")
	case "ANNOUNCE":
		return quorumpb.TrxType_ANNOUNCE, nil
	case "REQ_BLOCK_FORWARD":
		return quorumpb.TrxType_REQ_BLOCK_FORWARD, nil
	case "REQ_BLOCK_BACKWARD":
		return quorumpb.TrxType_REQ_BLOCK_BACKWARD, nil
	case "BLOCK_SYNCED":
		return quorumpb.TrxType_BLOCK_SYNCED, nil
	case "BLOCK_PRODUCED":
		return quorumpb.TrxType_BLOCK_PRODUCED, nil
	case "USER":
		return -1, errors.New("this trx type can not be configured")
	case "ASK_PEERID":
		return quorumpb.TrxType_ASK_PEERID, nil
	case "CHAIN_CONFIG":
		return -1, errors.New("this trx type can not be configured")
	case "APP_CONFIG":
		return -1, errors.New("this trx type can not be configured")
	default:
		return -1, errors.New("Unsupported TrxType")
	}
}
