package handlers

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
)

type UpdConsensusResult struct {
	GroupId      string `json:"group_id" validate:"required,uuid4" example:"5ed3f9fe-81e2-450d-9146-7a329aac2b62"`
	Producers    []string
	FromEpoch    uint64 `json:"start_from_epoch" validate:"required" example:"100"`
	TrxEpochTick uint64 `json:"trx_epoch_tick" validate:"required" example:"100"`
	TrxId        string `json:"trx_id" validate:"required,uuid4" example:"6bff5556-4dc9-4cb6-a595-2181aaebdc26"`
	Failable     *int   `json:"failable_producers" validate:"required" example:"1"`
	Memo         string `json:"memo" example:"comment/remark"`
}

type UpdConsensusParam struct {
	GroupId             string   `json:"group_id" validate:"required,uuid4" example:"5ed3f9fe-81e2-450d-9146-7a329aac2b62"`
	ProducerPubkey      []string `from:"producer_pubkey" json:"producer_pubkey"  validate:"required" example:"CAISIQOxCH2yVZPR8t6gVvZapxcIPBwMh9jB80pDLNeuA5s8hQ=="`
	FromNewEpoch        uint64   `from:"start_from_epoch" json:"start_from_epoch" validate:"required" example:"100"`
	TrxEpochTick        uint64   `from:"trx_epoch_tick" json:"trx_epoch_tick" validate:"required" example:"100"`
	AgreementTickLength uint64   `from:"agreement_tick_length" json:"agreement_tick_length" validate:"required"`
	AgreementTickCount  uint64   `from:"agreement_tick_count" json:"agreement_tick_count" validate:"required"`
	Memo                string   `from:"memo" json:"memo" example:"comment/remark"`
}

func UpdConsensus(chainapidb def.APIHandlerIface, params *UpdConsensusParam) (*UpdConsensusResult, error) {
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
		if params.TrxEpochTick < 500 {
			return nil, errors.New("trx epoch tick length should be greater than 500(ms)")
		}

		trxId, err := group.UpdConsensus(params.ProducerPubkey, params.AgreementTickLength, params.AgreementTickCount, params.FromNewEpoch, params.TrxEpochTick)
		if err != nil {
			return nil, err
		}

		failable := (len(bundle) - 1) / 3 /* 3F < N */
		result := &UpdConsensusResult{
			TrxId:        trxId,
			GroupId:      group.Item.GroupId,
			Producers:    params.ProducerPubkey,
			Failable:     &failable,
			Memo:         params.Memo,
			FromEpoch:    params.FromNewEpoch,
			TrxEpochTick: params.TrxEpochTick,
		}

		return result, nil
	}
}
