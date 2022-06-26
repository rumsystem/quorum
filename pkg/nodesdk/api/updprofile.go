package nodesdkapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
	rumchaindata "github.com/rumsystem/rumchaindata/pkg/data"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type CustomValidatorProfile struct {
	Validator *validator.Validate
}

func (cv *CustomValidatorProfile) Validate(i interface{}) error {
	switch i.(type) {
	case *quorumpb.Activity:
		inputobj := i.(*quorumpb.Activity)
		if inputobj.Type != handlers.Update {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("unknown type of Actitity: %s, expect: %s", inputobj.Type, handlers.Update))
		}

		if inputobj.Person == nil || inputobj.Target == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("Person or Target is nil"))
		}

		if inputobj.Target.Type == handlers.Group {
			if inputobj.Target.Id == "" {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("Target Group must not be nil"))
			}

			if inputobj.Person.Name == "" && inputobj.Person.Image == nil && inputobj.Person.Wallet == nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("Person must have name or image fields"))
			}
		}
	default:
		if err := cv.Validator.Struct(i); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}
	return nil
}

func (h *NodeSDKHandler) UpdProfile(c echo.Context) (err error) {
	paramspb := new(quorumpb.Activity)
	if err = c.Bind(paramspb); err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	groupid := paramspb.Target.Id

	nodesdkGroupItem, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(groupid)
	if err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	if nodesdkGroupItem.Group.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
		return rumerrors.NewBadRequestError("NodeSDK can not post to private group, use ChainSDK instead")
	}

	if paramspb.Person.Image != nil {
		_, formatname, err := image.Decode(bytes.NewReader(paramspb.Person.Image.Content))
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}
		if fmt.Sprintf("image/%s", formatname) != strings.ToLower(paramspb.Person.Image.MediaType) {
			msg := fmt.Sprintf("image format don't match, mediatype is %s but the file is %s", strings.ToLower(paramspb.Person.Image.MediaType), fmt.Sprintf("image/%s", formatname))
			return rumerrors.NewBadRequestError(msg)
		}
	}

	trxFactory := &rumchaindata.TrxFactory{}
	trxFactory.Init(nodesdkctx.GetCtx().Version, nodesdkGroupItem.Group, nodesdkctx.GetCtx().Name, nodesdkctx.GetCtx())

	trx, err := trxFactory.GetPostAnyTrxWithKeyAlias(nodesdkGroupItem.SignAlias, paramspb.Person)
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

	err = httpClient.UpdApiServer(nodesdkGroupItem.ApiUrl)
	if err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	resultInBytes, err := httpClient.Post(GetPostTrxURI(groupId), itemBytes)
	if err != nil {
		return rumerrors.NewBadRequestError(err.Error())
	}

	return c.JSON(http.StatusOK, string(resultInBytes))
}
