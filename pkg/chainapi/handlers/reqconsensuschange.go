package handlers

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
)

type ReqConsensusChangeResult struct {
	GroupId   string   `json:"group_id" validate:"required,uuid4" example:"5ed3f9fe-81e2-450d-9146-7a329aac2b62"`
	ReqId     string   `json:"req_id"   validate:"required,uuid4" example:"5ed3f9fe-81e2-450d-9146-7a329aac2b62"`
	Nonce     uint64   `json:"nonce"    validate:"required"`
	Producers []string `json:"producers" validate:"required"`
	FromBlock uint64   `json:"from_block" validate:"required" example:"100"`
	FromEpoch uint64   `json:"from_epoch" validate:"required" example:"100"`
	Epoch     uint64   `json:"epoch" validate:"required" example:"100"`
	StartFrom int64    `json:"start_from"`
}

type ReqConsensusChangeParam struct {
	GroupId             string   `json:"group_id" validate:"required,uuid4" example:"5ed3f9fe-81e2-450d-9146-7a329aac2b62"`
	ProducerPubkey      []string `from:"producer_pubkey" json:"producer_pubkey"  validate:"required" example:"CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ=="`
	FromBlock           uint64   `from:"from_block" json:"from_block" validate:"required" example:"1000"`
	FromEpoch           uint64   `from:"from_epoch" json:"start_from_epoch" validate:"required" example:"100"`
	Epoch               uint64   `from:"epoch" json:"trx_epoch_tick" validate:"required" example:"100"`
	AgreementTickLength uint64   `from:"agreement_tick_length" json:"agreement_tick_length" validate:"required"`
	AgreementTickCount  uint64   `from:"agreement_tick_count" json:"agreement_tick_count" validate:"required"`
	Memo                string   `from:"memo" json:"memo" example:"comment/remark"`
}

func ReqConsensusChange(chainapidb def.APIHandlerIface, params *ReqConsensusChangeParam) (*ReqConsensusChangeResult, error) {
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
		//check len of pubkey list
		if len(params.ProducerPubkey) == 0 {
			return nil, errors.New("producer pubkey list empty")
		}

		//check if pubkeys pubkey list are unique
		bundle := make(map[string]bool)
		for _, producerPubkey := range params.ProducerPubkey {
			if ok := bundle[producerPubkey]; ok {
				return nil, fmt.Errorf("producer pubkey should be unique")
			}
			bundle[producerPubkey] = true
		}

		//check if pubkeys are announced
		for _, producerPubkey := range params.ProducerPubkey {
			if producerPubkey == group.Item.OwnerPubKey {
				//skip owner
				continue
			}

			isAnnounced, err := chainapidb.IsProducerAnnounced(group.GroupId, producerPubkey, group.Nodename)
			if err != nil {
				return nil, err
			}

			if !isAnnounced {
				return nil, fmt.Errorf("producer <%s> is not announced", producerPubkey)
			}
		}

		if params.AgreementTickLength < 1000 {
			return nil, errors.New("agreement tick length should be greater than 1000(ms)")
		}

		if params.AgreementTickCount < 10 {
			return nil, errors.New("agreement tick count should be greater than 10")
		}

		//check trx epoch tick length
		if params.Epoch < 500 {
			return nil, errors.New("trx epoch tick length should be greater than 500(ms)")
		}

		reqId, nonce, err := group.ReqChangeConsensus(params.ProducerPubkey, params.AgreementTickLength, params.AgreementTickCount, params.FromBlock, params.FromEpoch, params.Epoch)

		if err != nil {
			return nil, err
		}

		result := &ReqConsensusChangeResult{
			GroupId:   group.Item.GroupId,
			ReqId:     reqId,
			Nonce:     nonce,
			Producers: params.ProducerPubkey,
			FromBlock: params.FromBlock,
			FromEpoch: params.FromEpoch,
			Epoch:     params.Epoch,
			StartFrom: time.Now().UnixNano(),
		}

		return result, nil
	}
}
