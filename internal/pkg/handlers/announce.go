package handlers

import (
	"bytes"
	"encoding/hex"
	"errors"
	"time"

	"github.com/go-playground/validator/v10"
	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

type AnnounceResult struct {
	GroupId                string `json:"group_id" validate:"required"`
	AnnouncedSignPubkey    string `json:"sign_pubkey" validate:"required"`
	AnnouncedEncryptPubkey string `json:"encrypt_pubkey"`
	Type                   string `json:"type" validate:"required"`
	Action                 string `json:"action" validate:"required"`
	Sign                   string `json:"sign" validate:"required"`
	TrxId                  string `json:"trx_id" validate:"required"`
}

type AnnounceParam struct {
	GroupId string `from:"group_id"    json:"group_id"    validate:"required"`
	Action  string `from:"action"      json:"action"      validate:"required,oneof=add remove"`
	Type    string `from:"type"        json:"type"        validate:"required,oneof=user producer"`
	Memo    string `from:"memo"        json:"memo"        validate:"required"`
}

func AnnounceHandler(params *AnnounceParam) (*AnnounceResult, error) {
	validate := validator.New()

	if err := validate.Struct(params); err != nil {
		return nil, err
	}

	var item *quorumpb.AnnounceItem
	item = &quorumpb.AnnounceItem{}
	item.GroupId = params.GroupId

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[item.GroupId]; !ok {
		return nil, errors.New("Can not find group")
	} else {
		if params.Type == "user" {
			item.Type = quorumpb.AnnounceType_AS_USER
		} else if params.Type == "producer" {
			item.Type = quorumpb.AnnounceType_AS_PRODUCER
		} else {
			return nil, errors.New("Unknown type")
		}

		if params.Action == "add" {
			item.Action = quorumpb.ActionType_ADD
		} else if params.Action == "remove" {
			item.Action = quorumpb.ActionType_REMOVE
		} else {
			return nil, errors.New("Unknown type")
		}

		item.SignPubkey = group.Item.UserSignPubkey

		if item.Type == quorumpb.AnnounceType_AS_USER {
			encryptPubkey, err := nodectx.GetNodeCtx().Keystore.GetEncodedPubkey(params.GroupId, localcrypto.Encrypt)
			if err != nil {
				return nil, err
			}
			item.EncryptPubkey = encryptPubkey
		}

		item.OwnerPubkey = ""
		item.OwnerSignature = ""
		item.Result = quorumpb.ApproveType_ANNOUNCED

		var buffer bytes.Buffer
		buffer.Write([]byte(item.GroupId))
		buffer.Write([]byte(item.SignPubkey))
		buffer.Write([]byte(item.EncryptPubkey))
		buffer.Write([]byte(item.Type.String()))
		hash := chain.Hash(buffer.Bytes())
		signature, err := nodectx.GetNodeCtx().Keystore.SignByKeyName(item.GroupId, hash)

		if err != nil {
			return nil, err
		}

		item.AnnouncerSignature = hex.EncodeToString(signature)
		item.TimeStamp = time.Now().UnixNano()
		item.Memo = params.Memo

		trxId, err := group.UpdAnnounce(item)

		if err != nil {
			return nil, err
		}

		var announceResult *AnnounceResult
		announceResult = &AnnounceResult{GroupId: item.GroupId, AnnouncedSignPubkey: item.SignPubkey, AnnouncedEncryptPubkey: item.EncryptPubkey, Type: item.Type.String(), Action: item.Action.String(), Sign: hex.EncodeToString(signature), TrxId: trxId}

		return announceResult, nil
	}
}
