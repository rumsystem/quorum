package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type SendTrxResult struct {
	TrxId   string `json:"trx_id"   validate:"required"`
	ErrInfo string `json:"err_info" validate:"required"`
}

type NodeSDKTrxItem struct {
	TrxData   []byte
	CipherKey string
}

func (h *Handler) SendTrx(c echo.Context) (err error) {

	output := make(map[string]string)
	trxItem := new(NodeSDKTrxItem)

	if err = c.Bind(trxItem); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	trx := new(quorumpb.Trx)
	err = proto.Unmarshal(trxItem.TrxData, trx)
	if err != nil {
		output[ERROR_INFO] = "UNMARSHAL_TRXDATA_FAILED"
		return c.JSON(http.StatusBadRequest, output)
	}

	fmt.Println(trx)

	//TBD
	//check payment/blocklist etc...

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[trx.GroupId]; ok {
		//private group is NOT supported
		if group.Item.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
			output[ERROR_INFO] = "FUNCTION_NOT_SUPPORTED"
			return c.JSON(http.StatusBadRequest, output)
		}

		//check if user can provide cipherkey
		if trxItem.CipherKey != group.Item.CipherKey {
			output[ERROR_INFO] = "INVALID_GROUP_CERTIFICATION"
			return c.JSON(http.StatusBadRequest, output)
		}

		trxId, err := group.SendTrx(trx, conn.ProducerChannel)
		var sendTrxResult *SendTrxResult
		if err != nil {
			sendTrxResult = &SendTrxResult{TrxId: trxId, ErrInfo: err.Error()}
			return c.JSON(http.StatusOK, sendTrxResult)
		} else {
			sendTrxResult = &SendTrxResult{TrxId: trxId, ErrInfo: "OK"}
			return c.JSON(http.StatusOK, sendTrxResult)
		}

	} else {
		output[ERROR_INFO] = "INVALID_GROUP"
		return c.JSON(http.StatusBadRequest, output)
	}
}
