package api

import (
	"bytes"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"github.com/labstack/echo/v4"

	chain "github.com/huo-ju/quorum/internal/pkg/chain"
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

	if group, ok := chain.GetNodeCtx().Groups[params.GroupId]; !ok {
		output[ERROR_INFO] = "Can not find group"
		return c.JSON(http.StatusBadRequest, output)
	} else if group.Item.OwnerPubKey != group.Item.UserSignPubkey {
		output[ERROR_INFO] = "Only group owner can add or remove producer"
		return c.JSON(http.StatusBadRequest, output)
	} else {

		item := &quorumpb.ProducerItem{}
		item.GroupId = params.GroupId
		item.ProducerPubkey = params.ProducerPubkey
		item.GroupOwnerPubkey = group.Item.OwnerPubKey

		var buffer bytes.Buffer
		buffer.Write([]byte(item.GroupId))
		buffer.Write([]byte(item.ProducerPubkey))
		buffer.Write([]byte(item.GroupOwnerPubkey))
		hash := chain.Hash(buffer.Bytes())

		ks := chain.GetNodeCtx().Keystore
		signature, err := ks.SignByKeyName(item.GroupId, hash)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		item.GroupOwnerSign = hex.EncodeToString(signature)
		item.Action = params.Action //add or remove
		item.Memo = params.Memo
		item.TimeStamp = time.Now().UnixNano()
		trxId, err := group.ChainCtx.UpdProducer(item)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		var blockGrpUserResult *GrpProducerResult
		blockGrpUserResult = &GrpProducerResult{GroupId: item.GroupId, ProducerPubkey: item.ProducerPubkey, OwnerPubkey: item.GroupOwnerPubkey, Sign: item.GroupOwnerSign, Action: item.Action, Memo: item.Memo, TrxId: trxId}

		return c.JSON(http.StatusOK, blockGrpUserResult)
	}
}