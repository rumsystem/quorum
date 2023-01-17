package handlers

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
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
	TrxId     string `json:"trx_id" validate:"required" example:"6bff5556-4dc9-4cb6-a595-2181aaebdc26"`
	GroupId   string `json:"group_id" validate:"required" example:"5ed3f9fe-81e2-450d-9146-7a329aac2b62"`
	Producers []*quorumpb.ProducerItem
	Failable  int    `json:"failable_producers" validate:"required" example:"1"`
	Memo      string `json:"memo" example:"comment/remark"`
}

type GrpProducerParam struct {
	ProducerPubkey []string `from:"producer_pubkey" json:"producer_pubkey"  validate:"required" example:"CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ==,CAISIQNaJGBzRlL6ApIqHduqfhA6T8VS52Am6MNFrlFLNICWdQ=="`
	GroupId        string   `json:"group_id" validate:"required" example:"5ed3f9fe-81e2-450d-9146-7a329aac2b62"`
	Memo           string   `from:"memo"            json:"memo" example:"comment/remark"`
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
		if len(params.ProducerPubkey) == 0 {
			return nil, errors.New("producer pubkey list empty")
		}

		//check if pubkey in producer list are unique
		bundle := make(map[string]bool)

		bftProducerBundle := &quorumpb.BFTProducerBundleItem{}
		producers := []*quorumpb.ProducerItem{}

		for _, producerPubkey := range params.ProducerPubkey {

			if ok := bundle[producerPubkey]; ok {
				return nil, errors.New(fmt.Errorf("producer pubkey should be unique").Error())
			}
			bundle[producerPubkey] = true

			isAnnounced, err := chainapidb.IsProducerAnnounced(group.Item.GroupId, producerPubkey, group.ChainCtx.GetNodeName())
			if err != nil {
				return nil, err
			}

			if !isAnnounced {
				return nil, errors.New(fmt.Errorf("producer %s is not announced", producerPubkey).Error())
			}

			producer, err := group.GetAnnouncedProducer(producerPubkey)
			if err != nil {
				return nil, err
			}

			if producer.Action == quorumpb.ActionType_REMOVE {
				return nil, errors.New(fmt.Errorf("can not add a non-active producer %s", producerPubkey).Error())
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
			item.Memo = params.Memo
			item.TimeStamp = time.Now().UnixNano()
			producers = append(producers, item)
		}

		bftProducerBundle.Producers = producers

		trxId, err := group.UpdProducerBundle(bftProducerBundle, sudo)
		if err != nil {
			return nil, err
		}

		totalProducers := len(bundle) + 1    /* owner*/
		failable := (totalProducers - 1) / 3 /* 3F < N */

		blockGrpUserResult := &GrpProducerResult{GroupId: group.Item.GroupId, Producers: bftProducerBundle.Producers, Failable: failable, Memo: params.Memo, TrxId: trxId}
		return blockGrpUserResult, nil
	}
}
