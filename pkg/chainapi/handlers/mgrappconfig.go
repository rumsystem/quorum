package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type AppConfigParam struct {
	Action  string `from:"action"   json:"action"   validate:"required,oneof=add del" example:"add"`
	GroupId string `from:"group_id" json:"group_id" validate:"required" example:"5ed3f9fe-81e2-450d-9146-7a329aac2b62"`
	Name    string `from:"name"     json:"name"     validate:"required" example:"test_bool"`
	Type    string `from:"type"     json:"type"     validate:"required,oneof=int string bool" example:"bool"`
	Value   string `from:"value"    json:"value"    validate:"required" example:"false"`
	Memo    string `from:"memo"     json:"memo" example:"add test_bool to group"`
}

type AppConfigResult struct {
	GroupId string `json:"group_id" validate:"required" example:"5ed3f9fe-81e2-450d-9146-7a329aac2b62"`
	Sign    string `json:"signature" validate:"required" example:"3045022100e1375e48cfbd51cb78afc413fcca084deae9eb7f8454c54832feb9ae00fada7702203ee6fe2292ea3a87d687ae3369012b7518010e555b913125b8a7bf54f211502a"`
	TrxId   string `json:"trx_id" validate:"required" example:"9e54c173-c1dd-429d-91fa-a6b43c14da77"`
}

func MgrAppConfig(params *AppConfigParam, sudo bool) (*AppConfigResult, error) {
	validate := validator.New()
	if err := validate.Struct(params); err != nil {
		return nil, err
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[params.GroupId]; !ok {
		return nil, rumerrors.ErrGroupNotFound
	} else if group.Item.OwnerPubKey != group.Item.UserSignPubkey {
		return nil, rumerrors.ErrOnlyGroupOwner
	} else {
		ks := nodectx.GetNodeCtx().Keystore

		base64key, err := ks.GetEncodedPubkey(params.GroupId, localcrypto.Sign)
		groupSignPubkey, err := base64.RawURLEncoding.DecodeString(base64key)
		//pubkeybytes, err := hex.DecodeString(hexkey)
		//p2ppubkey, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
		//groupSignPubkey, err := p2pcrypto.MarshalPublicKey(p2ppubkey)
		if err != nil {
			return nil, errors.New("group key can't be decoded, err:" + err.Error())
		}

		item := &quorumpb.AppConfigItem{}
		item.GroupId = params.GroupId
		if params.Action == "add" {
			item.Action = quorumpb.ActionType_ADD
		} else {
			item.Action = quorumpb.ActionType_REMOVE
		}
		item.Name = params.Name
		if params.Type == "bool" {
			item.Type = quorumpb.AppConfigType_BOOL
			_, err := strconv.ParseBool(params.Value)
			if err != nil {
				return nil, errors.New("type/value mismatch")
			}
		} else if params.Type == "int" {
			item.Type = quorumpb.AppConfigType_INT
			_, err := strconv.Atoi(params.Value)
			if err != nil {
				return nil, errors.New("type/value mismatch")
			}
		} else {
			item.Type = quorumpb.AppConfigType_STRING
		}

		item.Value = params.Value
		item.Memo = params.Memo

		var buffer bytes.Buffer
		buffer.Write([]byte(item.GroupId))
		buffer.Write([]byte(item.Action.String()))
		buffer.Write([]byte(item.Name))
		buffer.Write([]byte(item.Type.String()))
		buffer.Write([]byte(item.Value))
		buffer.Write([]byte(item.Memo))
		buffer.Write(groupSignPubkey)
		hash := localcrypto.Hash(buffer.Bytes())

		signature, err := ks.EthSignByKeyName(item.GroupId, hash)

		if err != nil {
			return nil, err
		}

		item.OwnerPubkey = group.Item.OwnerPubKey
		item.OwnerSign = hex.EncodeToString(signature)
		item.TimeStamp = time.Now().UnixNano()
		trxId, err := group.UpdAppConfig(item, sudo)

		if err != nil {
			return nil, err
		}

		result := &AppConfigResult{GroupId: item.GroupId, Sign: hex.EncodeToString(signature), TrxId: trxId}

		return result, nil
	}
}
