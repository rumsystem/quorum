package p2p

import (
	"bufio"

	guuid "github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-msgio/protoio"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/data/pb"
	"github.com/rumsystem/quorum/internal/pkg/storage"
)

type RexRelay struct {
	rex   *RexService
	dbmgr *storage.DbMgr
}

func NewRexRelay(rex *RexService, dbmgr *storage.DbMgr) *RexRelay {
	return &RexRelay{rex: rex, dbmgr: dbmgr}
}

func (r *RexRelay) Handler(rummsg *quorumpb.RumMsg, s network.Stream) error {
	if rummsg.MsgType == quorumpb.RumMsgType_RELAY_REQ && rummsg.RelayReq != nil {
		item := &quorumpb.GroupRelayItem{}
		item.GroupId = rummsg.RelayReq.GroupId
		item.UserPubkey = rummsg.RelayReq.UserPubkey
		item.Type = rummsg.RelayReq.Type
		item.Duration = rummsg.RelayReq.Duration
		item.SenderSign = rummsg.RelayReq.SenderSign
		item.Memo = rummsg.RelayReq.Memo
		item.ReqPeerId = s.Conn().RemotePeer().Pretty()
		r.dbmgr.AddRelayReq(item)

		//TEST write response
		bufw := bufio.NewWriter(s)
		wc := protoio.NewDelimitedWriter(bufw)
		err := wc.WriteMsg(rummsg)
		if err != nil {
			rumexchangelog.Debugf("writemsg to network stream err: %s", err)
		} else {
			rumexchangelog.Debugf("writemsg to network stream succ: %s.", "test")
		}
		bufw.Flush()

	} else if rummsg.MsgType == quorumpb.RumMsgType_RELAY_RESP && rummsg.RelayResp != nil {
		item := &quorumpb.GroupRelayItem{}
		item.RelayId = guuid.New().String()
		item.GroupId = rummsg.RelayResp.GroupId
		item.UserPubkey = rummsg.RelayResp.UserPubkey
		item.Type = rummsg.RelayResp.Type
		item.Duration = rummsg.RelayResp.Duration
		item.SenderSign = rummsg.RelayResp.SenderSign
		item.Memo = rummsg.RelayResp.Memo
		item.RelayPeerId = s.Conn().RemotePeer().Pretty()
		r.dbmgr.AddRelayActivity(item)
	}
	return nil
}
