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
	TrxId     string `json:"trx_id" validate:"required"`
	GroupId   string `json:"group_id" validate:"required"`
	Producers []*quorumpb.ProducerItem
	Memo      string `json:"memo"`
}

type GrpProducerParam struct {
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
			return nil, errors.New("for group use BFT consensus, you can only update group producers in even number(2,4,6...) each time")
		}

		bftProducerBundle := &quorumpb.BFTProducerBundleItem{}
		producers := []*quorumpb.ProducerItem{}

		for _, producerPubkey := range params.ProducerPubkey {

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

		blockGrpUserResult := &GrpProducerResult{GroupId: group.Item.GroupId, Producers: bftProducerBundle.Producers, Memo: params.Memo, TrxId: trxId}
		return blockGrpUserResult, nil
	}
}
