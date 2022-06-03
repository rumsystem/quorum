package nodesdkapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
	rumchaindata "github.com/rumsystem/rumchaindata/pkg/data"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type CustomValidatorPost struct {
	Validator *validator.Validate
}

func (cv *CustomValidatorPost) Validate(i interface{}) error {
	switch i.(type) {
	case *quorumpb.Activity:
		inputobj := i.(*quorumpb.Activity)
		if inputobj.Type == Add {
			if inputobj.Object != nil && inputobj.Target != nil {
				if inputobj.Target.Type == Group && inputobj.Target.Id != "" {
					if inputobj.Object.Type == Note && (inputobj.Object.Content != "" || len(inputobj.Object.Image) > 0) {
						return nil
					} else if inputobj.Object.Type == File && inputobj.Object.File != nil {
						return nil
					}
					return errors.New(fmt.Sprintf("unsupported object type: %s", inputobj.Object.Type))
				}
				return errors.New(fmt.Sprintf("Target Group must not be nil"))
			}
			return errors.New(fmt.Sprintf("Object and Target Object must not be nil"))
		} else if inputobj.Type == Like || inputobj.Type == Dislike {
			if inputobj.Object != nil && inputobj.Target != nil {
				if inputobj.Target.Type == Group && inputobj.Target.Id != "" {
					if inputobj.Object.Id != "" {
						return nil
					}
					return errors.New(fmt.Sprintf("unsupported object type: %s", inputobj.Object.Type))
				}
				return errors.New(fmt.Sprintf("Target Group must not be nil"))
			}
			return errors.New(fmt.Sprintf("Object and Target Object must not be nil"))
		}
		return errors.New(fmt.Sprintf("unknown type of Actitity: %s", inputobj.Type))
	default:
		if err := cv.Validator.Struct(i); err != nil {
			return err
		}
	}
	return nil
}

type SendTrxResult struct {
	TrxId   string `json:"trx_id"   validate:"required"`
	ErrInfo string `json:"err_info" validate:"required"`
}

func (h *NodeSDKHandler) PostToGroup() echo.HandlerFunc {
	return func(c echo.Context) error {
		var err error
		output := make(map[string]string)
		paramspb := new(quorumpb.Activity)

		if err = c.Bind(paramspb); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		validate := &CustomValidatorPost{Validator: validator.New()}
		if err := validate.Validate(paramspb); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		dbMgr := nodesdkctx.GetDbMgr()
		nodesdkGroupItem, err := dbMgr.GetGroupInfo(paramspb.Target.Id)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if nodesdkGroupItem.Group.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
			output[ERROR_INFO] = "NodeSDK can not post to private group, use ChainSDK instead"
			return c.JSON(http.StatusBadRequest, output)
		}

		trxFactory := &rumchaindata.TrxFactory{}
		trxFactory.Init(nodesdkctx.GetCtx().Version, nodesdkGroupItem.Group, nodesdkctx.GetCtx().Name, nodesdkctx.GetCtx())

		trx, err := trxFactory.GetPostAnyTrxWithKeyAlias(nodesdkGroupItem.SignAlias, paramspb.Object)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		trxBytes, err := proto.Marshal(trx)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		trxItem := new(NodeSDKTrxItem)
		trxItem.TrxBytes = trxBytes
		trxItem.JwtToken = JwtToken

		trxItemBytes, err := json.Marshal(trxItem)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		encryptData, err := getEncryptData(trxItemBytes, nodesdkGroupItem.Group.CipherKey)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		item := new(NodeSDKSendTrxItem)
		item.GroupId = nodesdkGroupItem.Group.GroupId
		item.TrxItem = encryptData

		itemBytes, err := json.Marshal(item)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		//just get the first one
		httpClient, err := nodesdkctx.GetCtx().GetHttpClient(nodesdkGroupItem.Group.GroupId)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		err = httpClient.UpdApiServer(nodesdkGroupItem.ApiUrl)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		resultInBytes, err := httpClient.Post(POST_TRX_URI, itemBytes)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		return c.JSON(http.StatusOK, string(resultInBytes))
	}
}
