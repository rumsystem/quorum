package consensus

import (
	guuid "github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func SendHbbRBC(groupId string, msg *quorumpb.BroadcastMsg, epoch int64, payloadType quorumpb.HBMsgPayloadType) error {
	connMgr, err := conn.GetConn().GetConnMgr(groupId)
	if err != nil {
		return err
	}

	msgB, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	hbmsg := &quorumpb.HBMsgv1{
		MsgId:       guuid.New().String(),
		MsgType:     quorumpb.HBBMsgType_BROADCAST,
		Epoch:       epoch,
		PayloadType: payloadType,
		Payload:     msgB,
	}

	return connMgr.SendHBMsg(hbmsg, conn.ProducerChannel)
}

func SendHbbAgreement(groupId string, msg *quorumpb.AgreementMsg, epoch int64, payloadType quorumpb.HBMsgPayloadType) error {
	connMgr, err := conn.GetConn().GetConnMgr(groupId)
	if err != nil {
		return err
	}

	msgB, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	hbmsg := &quorumpb.HBMsgv1{
		MsgId:       guuid.New().String(),
		MsgType:     quorumpb.HBBMsgType_AGREEMENT,
		Epoch:       epoch,
		PayloadType: payloadType,
		Payload:     msgB,
	}

	return connMgr.SendHBMsg(hbmsg, conn.ProducerChannel)
}
