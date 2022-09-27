package handlers

import (
	"bytes"
	"encoding/hex"
	"errors"
	"time"

	"github.com/go-playground/validator/v10"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

type GrpProducerResult struct {
	GroupId   string `json:"group_id" validate:"required"`
	Producers []*quorumpb.ProducerItem
	TrxId     string `json:"trx_id" validate:"required"`
	Memo      string `json:"memo"`
	Action    string `json:"action" validate:"required,oneof=ADD REMOVE"`
}

type GrpProducerParam struct {
	Action         string   `from:"action"          json:"action"           validate:"required,oneof=add remove"`
	ProducerPubkey []string `from:"producer_pubkey" json:"producer_pubkey"  validate:"required"`
	GroupId        string   `from:"group_id"        json:"group_id"         validate:"required"`
	Memo           string   `from:"memo"            json:"memo"`
}

func GroupProducer(chainapidb def.APIHandlerIface, params *GrpProducerParam, sudo bool) (*GrpProducerResult, error) {
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

		if len(params.ProducerPubkey)%2 != 0 {
			return nil, errors.New("For group use BFT consensus, you can only add/remove prodcers in even number(2,4,6...) each time")
		}

		bftProducerBundle := &quorumpb.BFTProducerBundleItem{}
		producers := []*quorumpb.ProducerItem{}

		for _, producerPubkey := range params.ProducerPubkey {

			isAnnounced, err := chainapidb.IsProducerAnnounced(group.Item.GroupId, producerPubkey, group.ChainCtx.GetNodeName())
			if err != nil {
				return nil, err
			}

			if !isAnnounced {
				return nil, errors.New("Producer is not announced")
			}

			producer, err := group.GetAnnouncedProducer(producerPubkey)
			if err != nil {
				return nil, err
			}

			if producer.Action == quorumpb.ActionType_REMOVE && params.Action == "add" {
				return nil, errors.New("Can not add a none active producer")
			}

			item := &quorumpb.ProducerItem{}
			item.GroupId = params.GroupId
			item.ProducerPubkey = producerPubkey
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
			producers = append(producers, item)
		}

		bftProducerBundle.Producers = producers

		trxId, err := group.UpdProducerBundle(bftProducerBundle, sudo)

		if err != nil {
			return nil, err
		}

		var blockGrpUserResult *GrpProducerResult
		blockGrpUserResult = &GrpProducerResult{GroupId: group.Item.GroupId, Producers: bftProducerBundle.Producers, Memo: params.Memo, TrxId: trxId}

		return blockGrpUserResult, nil
	}
}
