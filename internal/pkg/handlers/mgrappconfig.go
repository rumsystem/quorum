package handlers

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

type AppConfigParam struct {
	Action  string `from:"action"   json:"action"   validate:"required,oneof=add del"`
	GroupId string `from:"group_id" json:"group_id" validate:"required"`
	Name    string `from:"name"     json:"name"     validate:"required"`
	Type    string `from:"type"     json:"type"     validate:"required,oneof=int string bool"`
	Value   string `from:"value"    json:"value"    validate:"required"`
	Memo    string `from:"memo"     json:"memo"`
}

type AppConfigResult struct {
	GroupId string `json:"group_id" validate:"required"`
	Sign    string `json:"signature" validate:"required"`
	TrxId   string `json:"trx_id" validate:"required"`
}

func MgrAppConfig(params *AppConfigParam) (*AppConfigResult, error) {
	validate := validator.New()
	if err := validate.Struct(params); err != nil {
		return nil, err
	}

	ks := nodectx.GetNodeCtx().Keystore

	hexkey, err := ks.GetEncodedPubkey(params.GroupId, localcrypto.Sign)
	pubkeybytes, err := hex.DecodeString(hexkey)
	p2ppubkey, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
	groupSignPubkey, err := p2pcrypto.MarshalPublicKey(p2ppubkey)
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

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[item.GroupId]; !ok {
		return nil, errors.New("Can not find group")
	} else if group.Item.OwnerPubKey != group.Item.UserSignPubkey {
		return nil, errors.New("Only group owner can add or remove config of it")
	} else {
		var buffer bytes.Buffer
		buffer.Write([]byte(item.GroupId))
		buffer.Write([]byte(item.Action.String()))
		buffer.Write([]byte(item.Name))
		buffer.Write([]byte(item.Type.String()))
		buffer.Write([]byte(item.Value))
		buffer.Write([]byte(item.Memo))
		buffer.Write(groupSignPubkey)
		hash := chain.Hash(buffer.Bytes())

		signature, err := ks.SignByKeyName(item.GroupId, hash)

		if err != nil {
			return nil, err
		}

		item.OwnerPubkey = group.Item.OwnerPubKey
		item.OwnerSign = hex.EncodeToString(signature)
		item.TimeStamp = time.Now().UnixNano()
		trxId, err := group.UpdAppConfig(item)

		if err != nil {
			return nil, err
		}

		result := &AppConfigResult{GroupId: item.GroupId, Sign: hex.EncodeToString(signature), TrxId: trxId}

		return result, nil
	}
}
