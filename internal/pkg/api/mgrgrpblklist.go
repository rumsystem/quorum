package api

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/huo-ju/quorum/internal/pkg/chain"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"github.com/labstack/echo/v4"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

type CustomValidatorBklUsr struct {
	Validator *validator.Validate
}

func (cv *CustomValidatorBklUsr) Validate(i interface{}) error {
	switch i.(type) {
	case *quorumpb.Activity:
		inputobj := i.(*quorumpb.Activity)
		if inputobj.Type == Add || inputobj.Type == Remove {
			if inputobj.Object != nil && inputobj.Target != nil {
				if inputobj.Target.Type == Group && inputobj.Target.Id != "" {
					if inputobj.Object.Type == Auth && inputobj.Object.Id != "" {
						return nil
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

type BlockGrpUserResult struct {
	GroupId     string `json:"group_id"`
	UserId      string `json:"user_id"`
	OwnerPubkey string `json:"owner_pubkey"`
	Sign        string `json:"sign"`
	TrxId       string `json:"trx_id"`
	Memo        string `json:"memo"`
}

func (h *Handler) MgrGrpBlkList(c echo.Context) (err error) {
	output := make(map[string]string)
	paramspb := new(quorumpb.Activity)

	if err = c.Bind(paramspb); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	validate := &CustomValidatorBklUsr{Validator: validator.New()}

	if err = validate.Validate(paramspb); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	pubkeybytes, err := p2pcrypto.MarshalPublicKey(chain.GetChainCtx().PublicKey)
	var item *quorumpb.BlockListItem
	item = &quorumpb.BlockListItem{}
	item.UserId = paramspb.Object.Id
	item.GroupId = paramspb.Target.Id
	item.OwnerPubkey = p2pcrypto.ConfigEncodeKey(pubkeybytes)

	if group, ok := chain.GetChainCtx().Groups[item.GroupId]; !ok {
		output[ERROR_INFO] = "Can not find group"
		return c.JSON(http.StatusBadRequest, output)
	} else if group.Item.GetOwnerPubKey() != item.OwnerPubkey {
		output[ERROR_INFO] = "Only group owner can add or remove user to blocklist"
		return c.JSON(http.StatusBadRequest, output)
	} else {
		var buffer bytes.Buffer
		buffer.Write([]byte(item.GroupId))
		buffer.Write([]byte(item.UserId))
		buffer.Write(pubkeybytes)
		hash := chain.Hash(buffer.Bytes())
		signature, err := chain.Sign(hash)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		item.OwnerSign = fmt.Sprintf("%x", signature)
		item.TimeStamp = time.Now().UnixNano()
		item.Memo = paramspb.Type //add or remove

		trxId, err := group.UpdAuth(item)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		var blockGrpUserResult *BlockGrpUserResult
		blockGrpUserResult = &BlockGrpUserResult{GroupId: item.GroupId, UserId: item.UserId, OwnerPubkey: p2pcrypto.ConfigEncodeKey(pubkeybytes), Sign: fmt.Sprintf("%x", signature), Memo: item.Memo, TrxId: trxId}

		return c.JSON(http.StatusOK, blockGrpUserResult)
	}

}
