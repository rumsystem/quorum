package api

import (
	"encoding/json"
	"fmt"
	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	"github.com/labstack/echo/v4"
	"net/http"
)

func (h *Handler) PostToGroup(c echo.Context) (err error) {
	//should parse and check POST content
	//generate group protocols
	bodyBytes := []byte("{'ACTION'='POST_TO_GROUP', 'GROUP_ID'='test_group_id','CONTENT'='TEST ONLY CONTENT'}")
	var trx chain.Trx
	var trxMsg chain.TrxMsg

	trxMsg, _ = chain.CreateTrxMsgRegSign(chain.GetContext().PeerId.Pretty(), bodyBytes)

	trx.Msg = trxMsg
	trx.Data = bodyBytes

	var cons []string
	trx.Consensus = cons

	//save trx to local db
	chain.AddTrx(trx)

	jsonBytes, err := json.Marshal(trxMsg)
	if err != nil { // error json data
		return c.JSON(http.StatusOK, map[string]string{"create": fmt.Sprintf("%s", err)})
	}

	h.ChainCtx.PublicTopic.Publish(h.Ctx, jsonBytes)

	//return OK
	return c.JSON(http.StatusOK, map[string]int64{"post": 0})
}
