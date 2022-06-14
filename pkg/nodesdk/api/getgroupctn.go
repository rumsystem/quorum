package nodesdkapi

import (
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type GetGroupCtnPrarms struct {
	GroupId         string   `json:"group_id" validate:"required"`
	Num             int      `json:"num" validate:"required"`
	Nonce           string   `json:"nonce"`
	StartTrx        string   `json:"start_trx"`
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
	Content   proto.Message
	TypeUrl   string
	TimeStamp int64
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

		nodesdkGroupItem, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(params.GroupId)
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

		trxs := new([]*quorumpb.Trx)
		err = json.Unmarshal(resultInBytes, trxs)

		ctnobjList := []*GroupContentObjectItem{}
		for _, trx := range *trxs {

			//TODO: support private group
			//if item.TrxType == quorumpb.TrxType_POST && nodesdkGroupItem.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
			//	nodesdk not support private group now, encrypted by age for all announced group user
			//}
			//decrypt message by AES, for public group
			ciperKey, err := hex.DecodeString(nodesdkGroupItem.Group.CipherKey)
			if err != nil {
				return err
			}

			decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
			if err != nil {
				return err
			}
			ctnobj, typeurl, errum := quorumpb.BytesToMessage(trx.TrxId, decryptData)
			if errum != nil {
				c.Logger().Errorf("Unmarshal trx.Data %s Err: %s", trx.TrxId, errum)
			} else {
				ctnobjitem := &GroupContentObjectItem{TrxId: trx.TrxId, Publisher: trx.SenderPubkey, Content: ctnobj, TimeStamp: trx.TimeStamp, TypeUrl: typeurl}
				ctnobjList = append(ctnobjList, ctnobjitem)
			}
		}

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		return c.JSON(http.StatusOK, ctnobjList)
	}
}
