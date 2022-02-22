package handlers

import (
	"bytes"
	"encoding/hex"
	"errors"
	"time"

	"github.com/go-playground/validator/v10"
	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

type GrpProducerResult struct {
	GroupId        string `json:"group_id" validate:"required"`
	ProducerPubkey string `json:"producer_pubkey" validate:"required"`
	OwnerPubkey    string `json:"owner_pubkey" validate:"required"`
	Sign           string `json:"signature" validate:"required"`
	TrxId          string `json:"trx_id" validate:"required"`
	Memo           string `json:"memo"`
	Action         string `json:"action" validate:"required,oneof=ADD REMOVE"`
}

type GrpProducerParam struct {
	Action         string `from:"action"          json:"action"           validate:"required,oneof=add remove"`
	ProducerPubkey string `from:"producer_pubkey" json:"producer_pubkey"  validate:"required"`
	GroupId        string `from:"group_id"        json:"group_id"         validate:"required"`
	Memo           string `from:"memo"            json:"memo"`
}

func GroupProducer(params *GrpProducerParam) (*GrpProducerResult, error) {
	validate := validator.New()

	if err := validate.Struct(params); err != nil {
		return nil, err
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[params.GroupId]; !ok {
		return nil, errors.New("Can not find group")
	} else if group.Item.OwnerPubKey != group.Item.UserSignPubkey {
		return nil, errors.New("Only group owner can add or remove producer")
	} else {
		isAnnounced, err := group.IsProducerAnnounced(params.ProducerPubkey)
		if err != nil {
			return nil, err
		}

		if !isAnnounced {
			return nil, errors.New("Producer is not announced")
		}

		producer, err := group.GetAnnouncedProducer(params.ProducerPubkey)
		if err != nil {
			return nil, err
		}

		if producer.Action == quorumpb.ActionType_REMOVE && params.Action == "add" {
			return nil, errors.New("Can not add a none active producer")
		}

		item := &quorumpb.ProducerItem{}
		item.GroupId = params.GroupId
		item.ProducerPubkey = params.ProducerPubkey
		item.GroupOwnerPubkey = group.Item.OwnerPubKey

		var buffer bytes.Buffer
		buffer.Write([]byte(item.GroupId))
		buffer.Write([]byte(item.ProducerPubkey))
		buffer.Write([]byte(item.GroupOwnerPubkey))
		hash := chain.Hash(buffer.Bytes())

		ks := nodectx.GetNodeCtx().Keystore
		signature, err := ks.SignByKeyName(item.GroupId, hash)

		if err != nil {
			return nil, err
		}

		item.GroupOwnerSign = hex.EncodeToString(signature)
		if params.Action == "add" {
			item.Action = quorumpb.ActionType_ADD
		} else if params.Action == "remove" {
			item.Action = quorumpb.ActionType_REMOVE
		} else {
			return nil, errors.New("Unknown action")
		}

		item.Memo = params.Memo
		item.TimeStamp = time.Now().UnixNano()
		trxId, err := group.UpdProducer(item)

		if err != nil {
			return nil, err
		}

		var blockGrpUserResult *GrpProducerResult
		blockGrpUserResult = &GrpProducerResult{GroupId: item.GroupId, ProducerPubkey: item.ProducerPubkey, OwnerPubkey: item.GroupOwnerPubkey, Sign: item.GroupOwnerSign, Action: item.Action.String(), Memo: item.Memo, TrxId: trxId}

		return blockGrpUserResult, nil
	}
}
