package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

type ReqRelayParam struct {
	GroupId    string `from:"group_id" json:"group_id" validate:"required"`
	UserPubkey string `from:"user_pubkey" json:"user_pubkey"`
	RelayType  string `from:"relay_type" json:"relay_type" validate:"required,oneof=user group"`
	Duration   int64  `from:"duration" json:"duration" validate:"required"`
	SenderSign string `json:"signature" validate:"required"`
}

type RelayResult struct {
	Result bool `from:"result" json:"result" validate:"required"`
	//ReqId string `from:"req_id"      json:"req_id"      validate:"required"`
}

type RelayApproveResult struct {
	ReqId  string `from:"req_id"      json:"req_id"      validate:"required"`
	Result bool   `from:"result"      json:"result"      validate:"required"`
}

type RelayList struct {
	ReqList      []*quorumpb.GroupRelayItem `json:"req"`
	ApprovedList []*quorumpb.GroupRelayItem `json:"approved"`
	ActivityList []*quorumpb.GroupRelayItem `json:"activity"`
}

func (h *Handler) RequestRelay(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	input := new(ReqRelayParam)
	if err := cc.BindAndValidate(input); err != nil {
		return err
	}

	if input.RelayType == conn.RelayUserType || input.RelayType == conn.RelayGroupType {
		relayreq := quorumpb.RelayReq{}
		relayreq.GroupId = input.GroupId
		relayreq.UserPubkey = input.UserPubkey
		relayreq.Type = input.RelayType
		relayreq.Duration = input.Duration
		relayreq.SenderSign = input.SenderSign
		err := SendRelayRequestByRex(&relayreq)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}
		ret := RelayResult{true}
		return c.JSON(http.StatusOK, ret)
	} else {
		msg := fmt.Sprintf("unsupported relay type %s", input.RelayType)
		return rumerrors.NewBadRequestError(msg)
	}
}

func (h *Handler) ListRelay(c echo.Context) (err error) {
	reqresults, err := nodectx.GetNodeCtx().GetChainStorage().GetRelayReq("")
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	approvedresults, err := nodectx.GetNodeCtx().GetChainStorage().GetRelayApproved("")
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	activityresults, err := nodectx.GetNodeCtx().GetChainStorage().GetRelayActivity("")
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	ret := RelayList{ReqList: reqresults, ApprovedList: approvedresults, ActivityList: activityresults}
	return c.JSON(http.StatusOK, ret)
}

func (h *Handler) RemoveRelay(c echo.Context) (err error) {
	relayid := c.Param("relay_id")

	succ, relayitem, err := nodectx.GetNodeCtx().GetChainStorage().DeleteRelay(relayid)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	conn := conn.GetConn()
	conn.UnregisterChainRelay(relayid, relayitem.GroupId, relayitem.Type)
	ret := &RelayApproveResult{ReqId: relayid, Result: succ}

	return c.JSON(http.StatusOK, ret)
}

func (h *Handler) ApproveRelay(c echo.Context) (err error) {
	reqid := c.Param("req_id")

	succ, reqitem, err := nodectx.GetNodeCtx().GetChainStorage().ApproveRelayReq(reqid)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	if succ == true {
		conn := conn.GetConn()
		//add relay
		conn.RegisterChainRelay(reqitem.GroupId, reqitem.UserPubkey, reqitem.Type)
		relayresp := quorumpb.RelayResp{}
		relayresp.GroupId = reqitem.GroupId
		relayresp.UserPubkey = reqitem.UserPubkey
		relayresp.Type = reqitem.Type
		relayresp.Duration = reqitem.Duration
		relayresp.ApproveTime = time.Now().UnixNano()
		//send response
		SendRelayResponseByRex(&relayresp, reqitem.ReqPeerId)
	}
	ret := &RelayApproveResult{ReqId: reqid, Result: succ}
	return c.JSON(http.StatusOK, ret)
}

func SendRelayResponseByRex(relayresp *quorumpb.RelayResp, to string) error {
	rex := nodectx.GetNodeCtx().Node.RumExchange
	relayresp.RelayPeerId = []byte(rex.Host.ID())
	rummsg := &quorumpb.RumMsg{MsgType: quorumpb.RumMsgType_RELAY_RESP, RelayResp: relayresp}

	return rex.PublishToPeerId(rummsg, to)
}

func SendRelayRequestByRex(relayreq *quorumpb.RelayReq) error {
	rummsg := &quorumpb.RumMsg{MsgType: quorumpb.RumMsgType_RELAY_REQ, RelayReq: relayreq}
	succ := false
	rex := nodectx.GetNodeCtx().Node.RumExchange
	if rex != nil {
		for i := 0; i < 5; i++ { //try 5 peers
			err := rex.PublishToOneRandom(rummsg)
			if err == nil {
				succ = true
				break
			}
		}
	} else {
		return errors.New("RumExchange is nil, please set enablerumexchange as true")
	}
	if succ == false {
		return errors.New("failed publish to random peer ")
	}
	return nil
}

func SaveRelayRequest(input *ReqRelayParam) (string, error) {
	item := new(quorumpb.GroupRelayItem)
	item.GroupId = input.GroupId
	item.UserPubkey = input.UserPubkey
	item.Duration = input.Duration
	item.Type = input.RelayType
	item.SenderSign = input.SenderSign
	return nodectx.GetNodeCtx().GetChainStorage().AddRelayReq(item)
}
