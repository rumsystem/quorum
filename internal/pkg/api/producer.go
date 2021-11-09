package api

import (
	"bytes"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"

	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
)

type GrpProducerResult struct {
	GroupId        string `json:"group_id"`
	ProducerPubkey string `json:"producer_pubkey"`
	OwnerPubkey    string `json:"owner_pubkey"`
	Sign           string `json:"sign"`
	TrxId          string `json:"trx_id"`
	Memo           string `json:"memo"`
	Action         string `json:"action"`
}

type GrpProducerParam struct {
	Action         string `from:"action"          json:"action"           validate:"required,oneof=add remove"`
	ProducerPubkey string `from:"producer_pubkey" json:"producer_pubkey"  validate:"required"`
	GroupId        string `from:"group_id"        json:"group_id"         validate:"required"`
	Memo           string `from:"memo"            json:"memo"`
}

// @Tags Management
// @Summary AddProducer
// @Description add a peer to the group producer list
// @Accept json
// @Produce json
// @Param data body GrpProducerParam true "GrpProducerParam"
// @Success 200 {object} GrpProducerResult
// @Router /v1/group/producer [post]
func (h *Handler) GroupProducer(c echo.Context) (err error) {
	output := make(map[string]string)
	validate := validator.New()
	params := new(GrpProducerParam)

	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if err = validate.Struct(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[params.GroupId]; !ok {
		output[ERROR_INFO] = "Can not find group"
		return c.JSON(http.StatusBadRequest, output)
	} else if group.Item.OwnerPubKey != group.Item.UserSignPubkey {
		output[ERROR_INFO] = "Only group owner can add or remove producer"
		return c.JSON(http.StatusBadRequest, output)
	} else {
		isAnnounced, err := group.IsProducerAnnounced(params.ProducerPubkey)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if !isAnnounced {
			output[ERROR_INFO] = "Producer is not announced"
			return c.JSON(http.StatusBadRequest, output)
		}

		producer, err := group.GetAnnouncedProducer(params.ProducerPubkey)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if producer.Action == quorumpb.ActionType_REMOVE && params.Action == "add" {
			output[ERROR_INFO] = "Can not add a none active producer"
			return c.JSON(http.StatusBadRequest, output)
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
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		item.GroupOwnerSign = hex.EncodeToString(signature)
		if params.Action == "add" {
			item.Action = quorumpb.ActionType_ADD
		} else if params.Action == "remove" {
			item.Action = quorumpb.ActionType_REMOVE
		} else {
			output[ERROR_INFO] = "Unknown action"
			return c.JSON(http.StatusBadRequest, output)
		}

		item.Memo = params.Memo
		item.TimeStamp = time.Now().UnixNano()
		trxId, err := group.UpdProducer(item)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		var blockGrpUserResult *GrpProducerResult
		blockGrpUserResult = &GrpProducerResult{GroupId: item.GroupId, ProducerPubkey: item.ProducerPubkey, OwnerPubkey: item.GroupOwnerPubkey, Sign: item.GroupOwnerSign, Action: item.Action.String(), Memo: item.Memo, TrxId: trxId}

		return c.JSON(http.StatusOK, blockGrpUserResult)
	}
}
