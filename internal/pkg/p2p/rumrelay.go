package p2p

import (
	"github.com/libp2p/go-libp2p-core/network"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	//"google.golang.org/protobuf/proto"
	"fmt"
)

type RexRelay struct {
	rex   *RexService
	dbmgr *storage.DbMgr
}

func NewRexRelay(rex *RexService, dbmgr *storage.DbMgr) *RexRelay {
	return &RexRelay{rex: rex, dbmgr: dbmgr}
}

func (r *RexRelay) Handler(rummsg *quorumpb.RumMsg, s network.Stream) {
	if rummsg.MsgType == quorumpb.RumMsgType_RELAY_REQ && rummsg.RelayReq != nil {
		fmt.Println(rummsg)
		fmt.Println(r.dbmgr)
		item := &quorumpb.GroupRelayItem{}

		item.GroupId = rummsg.RelayReq.GroupId
		item.UserPubkey = rummsg.RelayReq.UserPubkey
		item.Type = rummsg.RelayReq.Type
		item.Duration = rummsg.RelayReq.Duration
		item.SenderSign = rummsg.RelayReq.SenderSign
		item.Memo = rummsg.RelayReq.Memo
		r.dbmgr.AddRelayReq(item)
	}
}
