package api

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/huo-ju/quorum/internal/pkg/chain"
	"github.com/huo-ju/quorum/internal/pkg/nodectx"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"github.com/labstack/echo/v4"
	//p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

type CustomValidatorSchema struct {
	Validator *validator.Validate
}

func (cv *CustomValidatorSchema) Validate(i interface{}) error {
	switch i.(type) {
	case *quorumpb.Activity:
		inputobj := i.(*quorumpb.Activity)
		if inputobj.Type == Add || inputobj.Type == Remove || inputobj.Type == Update {
			if inputobj.Object != nil && inputobj.Target != nil {
				if inputobj.Target.Type == Group && inputobj.Target.Id != "" {
					if inputobj.Object.Type == App {
						if inputobj.Object.Content != "" {
							return nil
						}
						return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("Schema added can not be empty"))
					}
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("unsupported object type: %s or Object Id can not be empty", inputobj.Object.Type))
				}
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("Target Group must not be nil"))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("Object and Target Object must not be nil"))
		}
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("unknown type of Actitity: %s", inputobj.Type))
	default:
		if err := cv.Validator.Struct(i); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}
	return nil
}

type SchemaResult struct {
	GroupId     string `json:"group_id"`
	OwnerPubkey string `json:"owner_pubkey"`
	Schema      string `json:"schema"`
	Sign        string `json:"sign"`
	TrxId       string `json:"trx_id"`
	Memo        string `json:"memo"`
}

func (h *Handler) Schema(c echo.Context) (err error) {

	output := make(map[string]string)
	paramspb := new(quorumpb.Activity)

	if err = c.Bind(paramspb); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	validate := &CustomValidatorSchema{Validator: validator.New()}

	if err = validate.Validate(paramspb); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	var item *quorumpb.SchemaItem
	item = &quorumpb.SchemaItem{}
	item.GroupId = paramspb.Target.Id
	item.SchemaJson = paramspb.Object.Content

	item.Memo = paramspb.Type

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[item.GroupId]; !ok {
		output[ERROR_INFO] = "Can not find group"
		return c.JSON(http.StatusBadRequest, output)
	} else if group.Item.OwnerPubKey != group.Item.UserSignPubkey {
		output[ERROR_INFO] = "Only group owner can add or remove schema to group"
		return c.JSON(http.StatusBadRequest, output)
	} else {
		item.GroupOwnerPubkey = group.Item.OwnerPubKey
		var buffer bytes.Buffer
		buffer.Write([]byte(item.GroupId))
		buffer.Write([]byte(item.SchemaJson))
		buffer.Write([]byte(item.GroupOwnerPubkey))
		buffer.Write([]byte(item.Memo))
		hash := chain.Hash(buffer.Bytes())

		//pbkeyByte, err := p2pcrypto.ConfigDecodeKey(item.GroupOwnerPubkey)
		//signature, err := chain.Sign(hash, pbkeyByte)
		ks := nodectx.GetNodeCtx().Keystore
		signature, err := ks.SignByKeyName(item.GroupId, hash)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		item.GroupOwnerSign = hex.EncodeToString(signature)
		item.TimeStamp = time.Now().UnixNano()
		trxId, err := group.ChainCtx.UpdSchema(item)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		blockGrpUserResult := &DenyUserResult{GroupId: item.GroupId, GroupOwnerPubkey: item.GroupOwnerPubkey, Sign: item.GroupOwnerSign, Memo: item.Memo, TrxId: trxId}

		return c.JSON(http.StatusOK, blockGrpUserResult)
	}
}
