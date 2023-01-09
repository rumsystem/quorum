package handlers

import (
	"bytes"
	"encoding/hex"
	"errors"
	"time"

	"github.com/go-playground/validator/v10"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type GrpProducerResult struct {
	GroupId        string `json:"group_id" validate:"required" example:"5ed3f9fe-81e2-450d-9146-7a329aac2b62"`
	ProducerPubkey string `json:"producer_pubkey" validate:"required" example:"CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ=="`
	OwnerPubkey    string `json:"owner_pubkey" validate:"required" example:"CAISIQNVGW0jrrKvo9/40lAyz/uICsyBbk465PmDKdWfcCM4JA=="`
	Sign           string `json:"signature" validate:"required" example:"304402202cbca750600cd0aeb3a1076e4aa20e9d1110fe706a553df90d0cd69289628eed022042188b48fa75d0197d9f5ce03499d3b95ffcdfb0ace707cf3eda9f12473db0ea"`
	TrxId          string `json:"trx_id" validate:"required" example:"6bff5556-4dc9-4cb6-a595-2181aaebdc26"`
	Memo           string `json:"memo" example:"comment/remark"`
	Action         string `json:"action" validate:"required,oneof=ADD REMOVE" example:"ADD"`
}

type GrpProducerParam struct {
	Action         string `from:"action"          json:"action"           validate:"required,oneof=add remove" example:"add"`
	ProducerPubkey string `from:"producer_pubkey" json:"producer_pubkey"  validate:"required" example:"CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ=="`
	GroupId        string `from:"group_id"        json:"group_id"         validate:"required" example:"5ed3f9fe-81e2-450d-9146-7a329aac2b62"`
	Memo           string `from:"memo"            json:"memo" example:"comment/remark"`
}

func GroupProducer(chainapidb def.APIHandlerIface, params *GrpProducerParam) (*GrpProducerResult, error) {
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
		isAnnounced, err := chainapidb.IsProducerAnnounced(group.Item.GroupId, params.ProducerPubkey, group.ChainCtx.GetNodeName())
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
		hash := localcrypto.Hash(buffer.Bytes())

		ks := nodectx.GetNodeCtx().Keystore
		signature, err := ks.EthSignByKeyName(item.GroupId, hash)

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
