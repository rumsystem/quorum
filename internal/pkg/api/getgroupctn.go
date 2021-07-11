package api

import (
	"fmt"
	"net/http"
	"strings"

	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"github.com/labstack/echo/v4"
	"google.golang.org/protobuf/proto"
)

type GroupContentObjectItem struct {
	TrxId     string
	Publisher string
	Content   *quorumpb.Object
	TimeStamp int64
}

type GroupContentPersonItem struct {
	TrxId     string
	Publisher string
	Content   *quorumpb.Person
	TimeStamp int64
}

func (h *Handler) GetGroupCtn(c echo.Context) (err error) {
	output := make(map[string]string)
	filter := strings.ToLower(c.QueryParam("filter"))
	groupid := c.Param("group_id")
	if groupid == "" {
		output[ERROR_INFO] = "group_id can't be nil."
		return c.JSON(http.StatusBadRequest, output)
	}

	if group, ok := chain.GetChainCtx().Groups[groupid]; ok {
		ctnList, err := chain.GetDbMgr().GetGrpCtnt(group.Item.GroupId, filter)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if filter == "profile" {
			var ctnobjList []*GroupContentPersonItem
			for _, ctn := range ctnList {
				ctnobj := &quorumpb.Person{}
				err = proto.Unmarshal(ctn.Content, ctnobj)
				if err == nil {
					ctnobjitem := &GroupContentPersonItem{TrxId: ctn.TrxId, Publisher: ctn.Publisher, Content: ctnobj, TimeStamp: ctn.TimeStamp}
					ctnobjList = append(ctnobjList, ctnobjitem)
				}
			}
			return c.JSON(http.StatusOK, ctnobjList)
		} else {
			var ctnobjList []*GroupContentObjectItem
			for _, ctn := range ctnList {
				ctnobj := &quorumpb.Object{}
				err = proto.Unmarshal(ctn.Content, ctnobj)
				if err == nil {
					ctnobjitem := &GroupContentObjectItem{TrxId: ctn.TrxId, Publisher: ctn.Publisher, Content: ctnobj, TimeStamp: ctn.TimeStamp}
					ctnobjList = append(ctnobjList, ctnobjitem)
				}
			}
			return c.JSON(http.StatusOK, ctnobjList)
		}
	} else {
		output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", groupid)
		return c.JSON(http.StatusBadRequest, output)
	}
}
