package api

import (
	"bytes"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/chain"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

type CustomValidatorSchema struct {
	Validator *validator.Validate
}

type SchemaParam struct {
	GroupId string `from:"group_id"    json:"group_id"    validate:"required"`
	Action  string `from:"action"      json:"action"      validate:"required,oneof=add remove"`
	Type    string `from:"type"        json:"type"        validate:"required"`
	Rule    string `from:"rule"	       json:"rule"        validate:"required"`
	Memo    string `from:"memo"        json:"memo"        validate:"required"`
}

type SchemaResult struct {
	GroupId     string `json:"group_id" validate:"required"`
	OwnerPubkey string `json:"owner_pubkey" validate:"required"`
	SchemaType  string `json:"schema_type" validate:"required"`
	SchemaRule  string `json:"schema_rule" validate:"required"`
	Action      string `json:"action" validate:"required"`
	Sign        string `json:"sign" validate:"required"`
	TrxId       string `json:"trx_id" validate:"required"`
}

// @Tags AppConfig
// @Summary Schema
// @Description Add schema to group
// @Accept json
// @Produce json
// @Param data body SchemaParam true "schema param"
// @Success 200 {object} SchemaResult
// @Router /api/v1/group/schema [post]
func (h *Handler) Schema(c echo.Context) (err error) {

	output := make(map[string]string)
	validate := validator.New()
	params := new(SchemaParam)

	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if err = validate.Struct(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	var item *quorumpb.SchemaItem
	item = &quorumpb.SchemaItem{}
	item.GroupId = params.GroupId
	item.Type = params.Type
	item.Rule = params.Rule

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[item.GroupId]; !ok {
		output[ERROR_INFO] = "Can not find group"
		return c.JSON(http.StatusBadRequest, output)
	} else if group.Item.OwnerPubKey != group.Item.UserSignPubkey {
		output[ERROR_INFO] = "Only group owner can add or remove schema to group"
		return c.JSON(http.StatusBadRequest, output)
	} else {
		item.GroupOwnerPubkey = group.Item.OwnerPubKey

		if params.Action == "add" {
			item.Action = quorumpb.ActionType_ADD
		} else if params.Action == "remove" {
			item.Action = quorumpb.ActionType_REMOVE
		} else {
			output[ERROR_INFO] = "Unknown action"
			return c.JSON(http.StatusBadRequest, output)
		}

		var buffer bytes.Buffer
		buffer.Write([]byte(item.GroupId))
		buffer.Write([]byte(item.Type))
		buffer.Write([]byte(item.Rule))
		buffer.Write([]byte(item.GroupOwnerPubkey))
		hash := chain.Hash(buffer.Bytes())

		ks := nodectx.GetNodeCtx().Keystore
		signature, err := ks.SignByKeyName(item.GroupId, hash)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		item.GroupOwnerSign = hex.EncodeToString(signature)
		item.TimeStamp = time.Now().UnixNano()
		trxId, err := group.UpdSchema(item)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		schemaResult := &SchemaResult{GroupId: item.GroupId, OwnerPubkey: item.GroupOwnerPubkey, SchemaType: item.Type, SchemaRule: item.Rule, Action: item.Action.String(), Sign: item.GroupOwnerSign, TrxId: trxId}

		return c.JSON(http.StatusOK, schemaResult)
	}
}
