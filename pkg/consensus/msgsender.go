package consensus

import (
	guuid "github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func SendHBBlockRBCMsg(groupId string, msg *quorumpb.RBCMsg, epoch int64) error {
	connMgr, err := conn.GetConn().GetConnMgr(groupId)
	if err != nil {
		return err
	}

	msgB, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	rbc := &quorumpb.HBBlockMsg{
		MsgType: quorumpb.HBBlockMsgType_RBC,
		Payload: msgB,
	}

	rbcb, err := proto.Marshal(rbc)
	if err != nil {
		return err
	}

	hbmsg := &quorumpb.HBMsgv1{
		MsgId:       guuid.New().String(),
		Epoch:       epoch,
		SessionId:   "",
		PayloadType: quorumpb.HBMsgPayloadType_HB_BLOCK,
		Payload:     rbcb,
	}

	return connMgr.BroadcastHBMsg(hbmsg)
}

func SendHBPSyncReqMsg(groupId string, msg *quorumpb.PSyncReq, epoch int64, sessionId string) error {
	connMgr, err := conn.GetConn().GetConnMgr(groupId)
	if err != nil {
		return err
	}

	msgB, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	psyncReq := &quorumpb.HBPSyncMsg{
		MsgType: quorumpb.PSyncMsgType_PSYNC_REQ,
		Payload: msgB,
	}

	syncb, err := proto.Marshal(psyncReq)
	if err != nil {
		return err
	}

	hbmsg := &quorumpb.HBMsgv1{
		MsgId:       guuid.New().String(),
		Epoch:       epoch,
		SessionId:   sessionId,
		PayloadType: quorumpb.HBMsgPayloadType_HB_PSYNC,
		Payload:     syncb,
	}

	return connMgr.BroadcastHBMsg(hbmsg)
}

func SendHBPSyncRespMsg(groupId string, msg *quorumpb.PSyncResp, epoch int64, sessionId string) error {
	connMgr, err := conn.GetConn().GetConnMgr(groupId)
	if err != nil {
		return err
	}

	msgB, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	psyncResp := &quorumpb.HBPSyncMsg{
		MsgType: quorumpb.PSyncMsgType_PSYNC_RESP,
		Payload: msgB,
	}

	syncb, err := proto.Marshal(psyncResp)
	if err != nil {
		return err
	}

	hbmsg := &quorumpb.HBMsgv1{
		MsgId:       guuid.New().String(),
		Epoch:       epoch,
		SessionId:   sessionId,
		PayloadType: quorumpb.HBMsgPayloadType_HB_PSYNC,
		Payload:     syncb,
	}

	return connMgr.BroadcastHBMsg(hbmsg)
}
