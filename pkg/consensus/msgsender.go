package consensus

import (
	guuid "github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func SendHBRBCMsg(groupId string, msg *quorumpb.RBCMsg, epoch int64) error {
	connMgr, err := conn.GetConn().GetConnMgr(groupId)
	if err != nil {
		return err
	}

	rbcb, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	hbmsg := &quorumpb.HBMsgv1{
		MsgId:       guuid.New().String(),
		Epoch:       epoch,
		PayloadType: quorumpb.HBMsgPayloadType_RBC,
		Payload:     rbcb,
	}

	return connMgr.BroadcastHBMsg(hbmsg)
}

func SendHBAABMsg(groupId string, msg quorumpb.BBAMsg, epoch int64) error {
	//TBD
	return nil
}

func SendPSyncReqMsg(groupId string, msg *quorumpb.PSyncReq) error {
	connMgr, err := conn.GetConn().GetConnMgr(groupId)
	if err != nil {
		return err
	}

	msgB, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	psyncReq := &quorumpb.PSyncMsg{
		MsgType: quorumpb.PSyncMsgType_PSYNC_REQ,
		Payload: msgB,
	}

	return connMgr.BroadcastPSyncMsg(psyncReq)
}

func SendPSyncRespMsg(groupId string, msg *quorumpb.PSyncResp) error {
	connMgr, err := conn.GetConn().GetConnMgr(groupId)
	if err != nil {
		return err
	}

	msgB, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	psyncResp := &quorumpb.PSyncMsg{
		MsgType: quorumpb.PSyncMsgType_PSYNC_RESP,
		Payload: msgB,
	}

	return connMgr.BroadcastPSyncMsg(psyncResp)
}
