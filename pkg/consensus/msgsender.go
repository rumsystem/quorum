package consensus

import (
	guuid "github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func SendHBRBCMsg(groupId string, msg *quorumpb.RBCMsg, epoch uint64) error {
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

func SendHBAABMsg(groupId string, msg *quorumpb.BBAMsg, epoch int64) error {
	//TBD
	return nil
}
