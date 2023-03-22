package handlers

import (
	"bytes"
	"encoding/hex"
	"errors"
	"time"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type AnnounceResult struct {
	GroupId                string `json:"group_id" validate:"required,uuid4" example:"17a598a0-274b-45e7-a4b5-b81f9f274d50"`
	AnnouncedSignPubkey    string `json:"sign_pubkey" validate:"required" example:"CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ=="`
	AnnouncedEncryptPubkey string `json:"encrypt_pubkey" example:"age1fx3ju9a2f3kpdh76375dect95wmvk084p8wxczeqdw8q2m0jtfks2k8pm9"`
	Type                   string `json:"type" validate:"required" example:"AS_USER"`
	Action                 string `json:"action" validate:"required" example:"ADD"`
	Sign                   string `json:"sign" validate:"required" example:"3046022100a853ca31f6f6719be213231b6428cecf64de5b1042dd8af1e140499507c85c40022100abd6828478f56da213ec10d361be8709333ff44cd0fa037409af9c0b67e6d0f5"`
	TrxId                  string `json:"trx_id" validate:"required,uuid4" example:"2e86c7fb-908e-4528-8f87-d3548e0137ab"`
}

type AnnounceParam struct {
	GroupId string `from:"group_id"    json:"group_id"    validate:"required,uuid4" example:"17a598a0-274b-45e7-a4b5-b81f9f274d50"`
	Action  string `from:"action"      json:"action"      validate:"required,oneof=add remove" example:"add"`
	Type    string `from:"type"        json:"type"        validate:"required,oneof=user producer" example:"user"`
	Memo    string `from:"memo"        json:"memo" example:"comment/remark"`
}

func AnnounceHandler(params *AnnounceParam) (*AnnounceResult, error) {
	item := &quorumpb.AnnounceItem{}
	item.GroupId = params.GroupId

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[item.GroupId]; !ok {
		return nil, rumerrors.ErrGroupNotFound
	} else {
		//check announce type according to node type, see document for more details
		if nodectx.GetNodeCtx().NodeType == nodectx.PRODUCER_NODE {
			if params.Type != "producer" {
				return nil, errors.New("Producer node can only announced as \"producer\"")
			}
		} else if nodectx.GetNodeCtx().NodeType == nodectx.FULL_NODE {
			if params.Type == "producer" {
				return nil, errors.New("Full node can not announce as producer")
			}
		}

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
			return nil, errors.New("Unknown action")
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
		hash := localcrypto.Hash(buffer.Bytes())
		signature, err := nodectx.GetNodeCtx().Keystore.EthSignByKeyName(item.GroupId, hash)

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

		announceResult := &AnnounceResult{
			GroupId:                item.GroupId,
			AnnouncedSignPubkey:    item.SignPubkey,
			AnnouncedEncryptPubkey: item.EncryptPubkey,
			Type:                   item.Type.String(),
			Action:                 item.Action.String(),
			Sign:                   hex.EncodeToString(signature),
			TrxId:                  trxId,
		}

		return announceResult, nil
	}
}
