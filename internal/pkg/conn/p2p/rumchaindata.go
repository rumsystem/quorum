package p2p

import (
	"fmt"

	"github.com/libp2p/go-libp2p/core/network"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type RexChainData struct {
	rex *RexService
}

func NewRexChainData(rex *RexService) *RexChainData {
	return &RexChainData{rex: rex}
}

func (r *RexChainData) Handler(rummsg *quorumpb.RumDataMsg, s network.Stream) error {
	frompeerid := s.Conn().RemotePeer()
	pkg := rummsg.DataPackage

	if pkg.Type == quorumpb.PackageType_SYNC_MSG {
		rumexchangelog.Debugf("receive a SYNC_MSG, from %s", frompeerid)
		syncMsg := &quorumpb.SyncMsg{}
		err := proto.Unmarshal(pkg.Data, syncMsg)
		if err == nil {
			targetchain, ok := r.rex.chainmgr[syncMsg.GroupId]
			if ok {
				return targetchain.HandleSyncMsgRex(syncMsg, s)
			} else {
				rumexchangelog.Warningf("receive a group unknown package, groupid: %s from: %s", syncMsg.GroupId, frompeerid)
			}
		} else {
			rumexchangelog.Warningf(err.Error())
		}
	} else {
		rumexchangelog.Warningf("receive a non syncMsg type package, %s", pkg.Type)
	}

	return fmt.Errorf("unsupported package type: %s", pkg.Type)
}
