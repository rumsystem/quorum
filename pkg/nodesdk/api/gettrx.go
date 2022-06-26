package nodesdkapi

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type Trx struct {
	TrxId        string
	Type         string
	GroupId      string
	Data         string
	TimeStamp    string
	Version      string
	Expired      string
	ResendCount  string
	Nonce        string
	SenderPubkey string
	SenderSign   string
	StorageType  string
}

const GET_TRX_URI string = "/api/v1/trx"

func (h *NodeSDKHandler) GetTrx() echo.HandlerFunc {
	return func(c echo.Context) error {
		groupid := c.Param("group_id")
		if groupid == "" {
			return rumerrors.NewBadRequestError("empty group id")
		}

		trxid := c.Param("trx_id")
		if trxid == "" {
			return rumerrors.NewBadRequestError("empty trx id")
		}

		nodesdkGroupItem, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(groupid)
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

		uri := GET_TRX_URI + "/" + groupid + "/" + trxid

		resultInBytes, err := httpClient.Get(uri)
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		trx := new(Trx)
		err = json.Unmarshal(resultInBytes, trx)
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		return c.JSON(http.StatusOK, trx)
	}
}
