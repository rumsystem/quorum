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
			return rumerrors.NewBadRequestError(err.Error())
		}

		validate := &CustomValidatorPost{Validator: validator.New()}
		if err := validate.Validate(paramspb); err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		nodesdkGroupItem, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(paramspb.Target.Id)
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		if nodesdkGroupItem.Group.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
			return rumerrors.NewBadRequestError(rumerrors.ErrEncryptionTypeNotSupported.Error())
		}

		trxFactory := &rumchaindata.TrxFactory{}
		trxFactory.Init(nodesdkctx.GetCtx().Version, nodesdkGroupItem.Group, nodesdkctx.GetCtx().Name, nodesdkctx.GetCtx())

		//assign type to paramspb.Object
		if paramspb.Object.Type == "" {
			paramspb.Object.Type = paramspb.Type
		}

		trx, err := trxFactory.GetPostAnyTrxWithKeyAlias(nodesdkGroupItem.SignAlias, paramspb.Object)
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		trxBytes, err := proto.Marshal(trx)
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		trxItem := new(NodeSDKTrxItem)
		trxItem.TrxBytes = trxBytes
		trxItem.JwtToken = JwtToken

		trxItemBytes, err := json.Marshal(trxItem)
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		encryptData, err := getEncryptData(trxItemBytes, nodesdkGroupItem.Group.CipherKey)
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		item := new(NodeSDKSendTrxItem)
		groupId := nodesdkGroupItem.Group.GroupId
		item.TrxItem = encryptData

		itemBytes, err := json.Marshal(item)
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		//just get the first one
		httpClient, err := nodesdkctx.GetCtx().GetHttpClient(nodesdkGroupItem.Group.GroupId)
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		if err = httpClient.UpdApiServer(nodesdkGroupItem.ApiUrl); err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		resultInBytes, err := httpClient.Post(GetPostTrxURI(groupId), itemBytes)
		res := TrxResult{}
		err = json.Unmarshal(resultInBytes, &res)
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		if res.TrxId == "" {
			errres := APIErrorResult{}
			err = json.Unmarshal(resultInBytes, &errres)
			if err != nil {
				return rumerrors.NewBadRequestError(err.Error())
			}
			return rumerrors.NewBadRequestError(errres.Error)
		}

		return c.JSON(http.StatusOK, res)
	}
}
