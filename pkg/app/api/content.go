package api

import (
	"net/http"
	"strconv"

	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"github.com/labstack/echo/v4"
	"google.golang.org/protobuf/proto"
)

type GroupContentObjectItem struct {
	TrxId     string
	Publisher string
	Content   proto.Message
	TypeUrl   string
	TimeStamp int64
}

type SenderList struct {
	Senders []string
}

// @Tags Apps
// @Summary GetGroupContents
// @Description Get contents in a group
// @Produce json
// @Param group_id path string  true "Group Id"
// @Param num query string false "the count of returns results"
// @Param reverse query boolean false "reverse = true will return results by most recently"
// @Param starttrx query string false "returns results from this trxid, but exclude it"
// @Param data body SenderList true "SenderList"
// @Success 200 {array} GroupContentObjectItem
// @Router /app/api/v1/group/{group_id}/content [post]
func (h *Handler) ContentByPeers(c echo.Context) (err error) {
	output := make(map[string]string)
	groupid := c.Param("group_id")
	num, _ := strconv.Atoi(c.QueryParam("num"))
	starttrx := c.QueryParam("starttrx")
	if num == 0 {
		num = 20
	}
	reverse := false
	if c.QueryParam("reverse") == "true" {
		reverse = true
	}
	senderlist := &SenderList{}
	if err = c.Bind(&senderlist); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}
	trxids, err := h.Appdb.GetGroupContentBySenders(groupid, senderlist.Senders, starttrx, num, reverse)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}
	ctnobjList := []*GroupContentObjectItem{}
	for _, trxid := range trxids {
		trx, err := h.Chaindb.GetTrx(trxid, h.NodeName)
		if err != nil {
			c.Logger().Errorf("GetTrx Err: %s", err)
			continue
		}
		ctnobj, typeurl, err := quorumpb.BytesToMessage(trx.TrxId, trx.Data)
		if err != nil {
			c.Logger().Errorf("Unmarshal trx.Data %s Err: %s", trx.TrxId, err)
		}
		ctnobjitem := &GroupContentObjectItem{TrxId: trx.TrxId, Publisher: trx.SenderPubkey, Content: ctnobj, TimeStamp: trx.TimeStamp, TypeUrl: typeurl}
		ctnobjList = append(ctnobjList, ctnobjitem)
	}
	return c.JSON(http.StatusOK, ctnobjList)
}
