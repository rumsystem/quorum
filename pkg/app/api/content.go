package api

import (
	"encoding/json"
	"fmt"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"github.com/labstack/echo/v4"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"net/http"
	"strconv"
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

func (h *Handler) Content(c echo.Context) (err error) {
	output := make(map[string]string)
	groupid := c.Param("group_id")
	num, _ := strconv.Atoi(c.QueryParam("num"))
	start, err := strconv.ParseInt(c.QueryParam("start"), 10, 64)
	if num == 0 {
		num = 20
	}

	trxids, err := h.Appdb.GetGroupContent(groupid, uint64(start), num)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}
	var ctnobjList []*GroupContentObjectItem
	for _, trxid := range trxids {
		apiurl := fmt.Sprintf("%s/trx/%s", h.Apiroot, trxid)
		req, err := http.NewRequest("GET", apiurl, nil)
		if err != nil {
			c.Logger().Errorf("request %s Err: %s", apiurl, err)
			continue
		}
		req.Header.Add("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.Logger().Errorf("request %s Err: %s", apiurl, err)
			continue
		}
		if resp.StatusCode == 200 {
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				c.Logger().Errorf("read %s Err: %s", apiurl, err)
				continue
			}
			var trx quorumpb.Trx
			err = json.Unmarshal(body, &trx)
			if err != nil {
				c.Logger().Errorf("Unmarshal %s Err: %s", apiurl, err)
				continue
			}
			ctnobj, typeurl, err := quorumpb.BytesToMessage(trx.TrxId, trx.Data)
			if err != nil {
				c.Logger().Errorf("Unmarshal trx.Data %s Err: %s", apiurl, err)
				continue
			}
			ctnobjitem := &GroupContentObjectItem{TrxId: trx.TrxId, Publisher: trx.Sender, Content: ctnobj, TimeStamp: trx.TimeStamp, TypeUrl: typeurl}
			ctnobjList = append(ctnobjList, ctnobjitem)

		} else {
			output[ERROR_INFO] = resp.Status
			return c.JSON(http.StatusBadRequest, output)
		}
	}

	return c.JSON(http.StatusOK, ctnobjList)
}

func (h *Handler) ContentByPeers(c echo.Context) (err error) {
	output := make(map[string]string)
	groupid := c.Param("group_id")
	num, _ := strconv.Atoi(c.QueryParam("num"))
	start, err := strconv.ParseInt(c.QueryParam("start"), 10, 64)
	if num == 0 {
		num = 20
	}
	senderlist := &SenderList{}
	if err = c.Bind(&senderlist); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}
	trxids, err := h.Appdb.GetGroupContentBySenders(groupid, senderlist.Senders, uint64(start), num)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}
	var ctnobjList []*GroupContentObjectItem
	for _, trxid := range trxids {
		apiurl := fmt.Sprintf("%s/trx/%s", h.Apiroot, trxid)
		req, err := http.NewRequest("GET", apiurl, nil)
		if err != nil {
			c.Logger().Errorf("request %s Err: %s", apiurl, err)
			continue
		}
		req.Header.Add("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.Logger().Errorf("request %s Err: %s", apiurl, err)
			continue
		}
		if resp.StatusCode == 200 {
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				c.Logger().Errorf("read %s Err: %s", apiurl, err)
				continue
			}
			var trx quorumpb.Trx
			err = json.Unmarshal(body, &trx)
			if err != nil {
				c.Logger().Errorf("Unmarshal %s Err: %s", apiurl, err)
				continue
			}
			ctnobj, typeurl, err := quorumpb.BytesToMessage(trx.TrxId, trx.Data)
			if err != nil {
				c.Logger().Errorf("Unmarshal trx.Data %s Err: %s", apiurl, err)
				continue
			}
			ctnobjitem := &GroupContentObjectItem{TrxId: trx.TrxId, Publisher: trx.Sender, Content: ctnobj, TimeStamp: trx.TimeStamp, TypeUrl: typeurl}
			ctnobjList = append(ctnobjList, ctnobjitem)

		} else {
			output[ERROR_INFO] = resp.Status
			return c.JSON(http.StatusBadRequest, output)
		}
	}
	return c.JSON(http.StatusOK, ctnobjList)
}
