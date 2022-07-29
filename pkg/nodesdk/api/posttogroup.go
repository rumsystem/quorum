package nodesdkapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
	rumchaindata "github.com/rumsystem/rumchaindata/pkg/data"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type CustomValidatorPost struct {
	Validator *validator.Validate
}

type TrxResult struct {
	TrxId string `json:"trx_id" validate:"required"`
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

func (h *NodeSDKHandler) PostToGroup() echo.HandlerFunc {
	return func(c echo.Context) error {
		paramspb := new(quorumpb.Activity)

		if err := c.Bind(paramspb); err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		validate := &CustomValidatorPost{Validator: validator.New()}
		if err := validate.Validate(paramspb); err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		nodesdkGroupItem, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(paramspb.Target.Id)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		if nodesdkGroupItem.Group.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
			return rumerrors.NewBadRequestError(rumerrors.ErrEncryptionTypeNotSupported)
		}

		trxFactory := &rumchaindata.TrxFactory{}
		trxFactory.Init(nodesdkctx.GetCtx().Version, nodesdkGroupItem.Group, nodesdkctx.GetCtx().Name, nodesdkctx.GetCtx())

		//assign type to paramspb.Object
		if paramspb.Object.Type == "" {
			paramspb.Object.Type = paramspb.Type
		}

		trx, err := trxFactory.GetPostAnyTrx(nodesdkGroupItem.SignAlias, paramspb.Object)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		trxBytes, err := proto.Marshal(trx)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		trxItem := new(NodeSDKTrxItem)
		trxItem.TrxBytes = trxBytes

		trxItemBytes, err := json.Marshal(trxItem)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		encryptData, err := getEncryptData(trxItemBytes, nodesdkGroupItem.Group.CipherKey)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		item := new(NodeSDKSendTrxItem)
		groupId := nodesdkGroupItem.Group.GroupId
		item.TrxItem = encryptData

		//just get the first one
		httpClient, err := nodesdkctx.GetCtx().GetHttpClient(nodesdkGroupItem.Group.GroupId)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		if err = httpClient.UpdApiServer(nodesdkGroupItem.ApiUrl); err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		res := new(TrxResult)
		err = httpClient.RequestChainAPI(GetPostTrxURI(groupId), http.MethodPost, item, nil, res)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		return c.JSON(http.StatusOK, res)
	}
}
