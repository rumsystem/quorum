package nodesdkapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type GetGroupCtnPrarms struct {
	GroupId         string   `json:"group_id" validate:"required"`
	Num             int      `json:"num" validate:"required"`
	Nonce           string   `json:"nonce"`
	StartTrx        string   `json:"start_trx" validate:"required"`
	Reverse         string   `json:"reverse" validate:"required,oneof=true false"`
	IncludeStartTrx string   `json:"include_start_trx" validate:"required,oneof=true false"`
	Senders         []string `json:"senders"`
}

type GetGroupCtnItem struct {
	Req      *GetGroupCtnPrarms
	JwtToken string
}

type GetGroupCtnReqItem struct {
	GroupId string
	Req     []byte
}

type GroupContentObjectItem struct {
	TrxId     string
	Publisher string
	Content   []byte
	TimeStamp int64
}

type GetGroupCtnResult struct {
	Contents []*GroupContentObjectItem `json:"contents"`
}

func (h *NodeSDKHandler) GetGroupCtn() echo.HandlerFunc {
	return func(c echo.Context) error {
		var err error
		output := make(map[string]string)

		validate := validator.New()
		params := new(GetGroupCtnPrarms)
		if err = c.Bind(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if err = validate.Struct(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		dbMgr := nodesdkctx.GetDbMgr()
		nodesdkGroupItem, err := dbMgr.GetGroupInfo(params.GroupId)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		getGroupCtnItem := new(GetGroupCtnItem)
		getGroupCtnItem.Req = params
		getGroupCtnItem.JwtToken = JwtToken

		itemBytes, err := json.Marshal(getGroupCtnItem)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		encryptData, err := getEncryptData(itemBytes, nodesdkGroupItem.Group.CipherKey)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		getGroupCtnReqItem := new(GetGroupCtnReqItem)
		getGroupCtnReqItem.GroupId = params.GroupId
		getGroupCtnReqItem.Req = encryptData

		reqBytes, err := json.Marshal(getGroupCtnReqItem)
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

		resultInBytes, err := httpClient.Post(GET_CTN_URI, reqBytes)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		result := string(resultInBytes)
		return c.JSON(http.StatusOK, result)
	}
}
