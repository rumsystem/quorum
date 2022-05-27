package nodesdkapi

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
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
		var err error
		output := make(map[string]string)

		groupid := c.Param("group_id")
		if groupid == "" {
			output[ERROR_INFO] = "group_id can't be nil."
			return c.JSON(http.StatusBadRequest, output)
		}

		trxid := c.Param("trx_id")
		if trxid == "" {
			output[ERROR_INFO] = "trx_id can't be nil."
			return c.JSON(http.StatusBadRequest, output)
		}

		dbMgr := nodesdkctx.GetDbMgr()
		nodesdkGroupItem, err := dbMgr.GetGroupInfo(groupid)
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

		uri := GET_TRX_URI + "/" + groupid + "/" + trxid

		resultInBytes, err := httpClient.Get(uri)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		trx := new(Trx)
		err = json.Unmarshal(resultInBytes, trx)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}
		return c.JSON(http.StatusOK, trx)
	}
}
