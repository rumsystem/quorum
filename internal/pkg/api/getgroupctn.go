package api

import (
	"fmt"
	"net/http"
	"strings"

	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"github.com/labstack/echo/v4"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type GroupContentObjectItem struct {
	TrxId     string
	Publisher string
	Content   proto.Message
	TypeUrl   string
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

		var ctnobjList []*GroupContentObjectItem
		for _, ctn := range ctnList {
			anyobj := &anypb.Any{}
			err := proto.Unmarshal(ctn.Content, anyobj)
			if err != nil {
				c.Logger().Debugf("Unmarshal Content %s Err: %s", ctn.TrxId, err)
			}
			var ctnobj proto.Message
			var typeurl string
			ctnobj, err = anyobj.UnmarshalNew()
			if err != nil { //old data pb.Object{} compatibility
				ctnobj = &quorumpb.Object{}
				err = proto.Unmarshal(ctn.Content, ctnobj)
				if err != nil {
					c.Logger().Debugf("try old data compatibility Unmarshal %s Err: %s", ctn.TrxId, err)
				} else {
					typeurl = "quorum.pb.Object"
				}
			} else {
				typeurl = strings.Replace(anyobj.TypeUrl, "type.googleapis.com/", "", 1)
			}
			if err == nil {
				ctnobjitem := &GroupContentObjectItem{TrxId: ctn.TrxId, Publisher: ctn.Publisher, Content: ctnobj, TimeStamp: ctn.TimeStamp, TypeUrl: typeurl}
				ctnobjList = append(ctnobjList, ctnobjitem)
			}
		}
		return c.JSON(http.StatusOK, ctnobjList)
	} else {
		output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", groupid)
		return c.JSON(http.StatusBadRequest, output)
	}
}
