package p2p

import (
	"bufio"

	guuid "github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-msgio/protoio"
	csdef "github.com/rumsystem/quorum/internal/pkg/storage/def"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

type RexRelay struct {
	rex *RexService
	cs  csdef.ChainStorageIface
	//dbmgr *storage.DbMgr
}

func NewRexRelay(rex *RexService, cs csdef.ChainStorageIface) *RexRelay {
	return &RexRelay{rex: rex, cs: cs}
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
		r.cs.AddRelayReq(item)

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
		r.cs.AddRelayActivity(item)
	}
	return nil
}
