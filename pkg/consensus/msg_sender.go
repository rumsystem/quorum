package consensus

import (
	guuid "github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func SendHbbRBC(groupId string, msg *quorumpb.BroadcastMsg) error {
	connMgr, err := conn.GetConn().GetConnMgr(groupId)
	if err != nil {
		return err
	}

	msgB, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	hbmsg := &quorumpb.HBMsg{
		MsgId:   guuid.New().String(),
		MsgType: quorumpb.HBBMsgType_BROADCAST,
		Payload: msgB,
	}

	return connMgr.SendHBMsg(hbmsg, conn.ProducerChannel)
}

func SendHbbAgreement(groupId string, msg *quorumpb.AgreementMsg) error {
	connMgr, err := conn.GetConn().GetConnMgr(groupId)
	if err != nil {
		return err
	}

	msgB, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	hbmsg := &quorumpb.HBMsg{
		MsgId:   guuid.New().String(),
		MsgType: quorumpb.HBBMsgType_AGREEMENT,
		Payload: msgB,
	}

	return connMgr.SendHBMsg(hbmsg, conn.ProducerChannel)
}
