package api

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"net/http"
)

type ReqRelayParam struct {
	GroupId    string `from:"group_id"      json:"group_id"      validate:"required"`
	UserPubkey string `from:"user_pubkey"      json:"user_pubkey"`
	RelayType  string `from:"relay_type"  json:"relay_type"  validate:"required,oneof=user group"`
	Duration   int64  `from:"duration"  json:"duration"  validate:"required"`
	SenderSign string `json:"signature" validate:"required"`
}

type RelayResult struct {
	ReqId string `from:"req_id"      json:"req_id"      validate:"required"`
}

type RelayApproveResult struct {
	ReqId  string `from:"req_id"      json:"req_id"      validate:"required"`
	Result bool   `from:"result"      json:"result"      validate:"required"`
}

type RelayList struct {
	ReqList      []*quorumpb.GroupRelayItem `json:"req"`
	ApprovedList []*quorumpb.GroupRelayItem `json:"approved"`
}

func (h *Handler) RequestRelayTest(c echo.Context) (err error) {
	var input ReqRelayParam
	output := make(map[string]string)

	if err = c.Bind(&input); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}
	if input.RelayType == conn.RelayUserType || input.RelayType == conn.RelayGroupType {
		reqid, err := SaveRelayRequest(&input)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}
		ret := RelayResult{reqid}
		return c.JSON(http.StatusOK, ret)
	} else {
		output[ERROR_INFO] = fmt.Sprintf("unsupported relay type %s", input.RelayType)
		return c.JSON(http.StatusBadRequest, output)
	}
}

func (h *Handler) ListRelay(c echo.Context) (err error) {
	output := make(map[string]string)
	reqresults, err := nodectx.GetDbMgr().GetRelayReq("")
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}
	approvedresults, err := nodectx.GetDbMgr().GetRelayApproved("")
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	ret := RelayList{ReqList: reqresults, ApprovedList: approvedresults}
	return c.JSON(http.StatusOK, ret)
}

func (h *Handler) RemoveRelay(c echo.Context) (err error) {
	output := make(map[string]string)
	relayid := c.Param("relay_id")
	succ, err := nodectx.GetDbMgr().DeleteRelay(relayid)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}
	ret := &RelayApproveResult{ReqId: relayid, Result: succ}
	return c.JSON(http.StatusOK, ret)
}

func (h *Handler) ApproveRelay(c echo.Context) (err error) {
	output := make(map[string]string)
	reqid := c.Param("req_id")
	succ, reqitem, err := nodectx.GetDbMgr().ApproveRelayReq(reqid)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}
	if succ == true {
		conn := conn.GetConn()
		conn.RegisterChainRelay(reqitem.GroupId, reqitem.UserPubkey, reqitem.Type)
		//add relay
	}
	ret := &RelayApproveResult{ReqId: reqid, Result: succ}
	return c.JSON(http.StatusOK, ret)
}

func SaveRelayRequest(input *ReqRelayParam) (string, error) {
	item := new(quorumpb.GroupRelayItem)
	item.GroupId = input.GroupId
	item.UserPubkey = input.UserPubkey
	item.Duration = input.Duration
	item.Type = input.RelayType
	item.SenderSign = input.SenderSign
	return nodectx.GetDbMgr().AddRelayReq(item)
}
