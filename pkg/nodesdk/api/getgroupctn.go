package nodesdkapi

import (
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
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
	Req []byte
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
		cc := c.(*utils.CustomContext)
		params := new(GetGroupCtnPrarms)
		if err := cc.BindAndValidate(params); err != nil {
			return err
		}

		nodesdkGroupItem, err := nodesdkctx.GetCtx().GetChainStorage().GetGroupInfoV2(params.GroupId)
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		getGroupCtnItem := new(GetGroupCtnItem)
		getGroupCtnItem.Req = params
		getGroupCtnItem.JwtToken = JwtToken

		itemBytes, err := json.Marshal(getGroupCtnItem)
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		encryptData, err := getEncryptData(itemBytes, nodesdkGroupItem.Group.CipherKey)
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		getGroupCtnReqItem := new(GetGroupCtnReqItem)
		groupId := params.GroupId
		getGroupCtnReqItem.Req = encryptData

		reqBytes, err := json.Marshal(getGroupCtnReqItem)
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		//just get the first one
		httpClient, err := nodesdkctx.GetCtx().GetHttpClient(nodesdkGroupItem.Group.GroupId)
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		if err := httpClient.UpdApiServer(nodesdkGroupItem.ApiUrl); err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		resultInBytes, err := httpClient.Post(GetGroupCtnURI(groupId), reqBytes)
		if err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

		trxs := new([]*quorumpb.Trx)
		if err := json.Unmarshal(resultInBytes, trxs); err != nil {
			return rumerrors.NewBadRequestError(err.Error())
		}

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

		return c.JSON(http.StatusOK, ctnobjList)
	}
}
