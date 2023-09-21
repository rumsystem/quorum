package p2p

import (
	"bytes"
	"fmt"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/rumsystem/quorum/internal/pkg/utils"
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

	if pkg.Type == quorumpb.PackageType_BLOCK_SYNC {
		//decompress syncmsg data
		content := new(bytes.Buffer)
		if err := utils.Decompress(bytes.NewReader(pkg.Data), content); err != nil {
			rumexchangelog.Errorf("utils.Decompress failed: <%s>", err)
			return fmt.Errorf("utils.Decompress failed: <%s>", err)
		}
		syncMsgByts := content.Bytes()
		syncMsg := &quorumpb.SyncMsg{}
		err := proto.Unmarshal(syncMsgByts, syncMsg)
		if err == nil {
			targetchain, ok := r.rex.chainmgr[syncMsg.GroupId]
			if ok {
				rumexchangelog.Debugf("receive Sync Msg, peerId <%s>, groupId <%s>, handle it", frompeerid, syncMsg.GroupId)
				return targetchain.HandleSyncMsgRex(syncMsg, s)
			} else {
				rumexchangelog.Debugf("receive Sync Msg, peerId <%s>, groupId <%s> not found, ignore", frompeerid, syncMsg.GroupId)
			}
		} else {
			rumexchangelog.Warningf(err.Error())
		}
	} else {
		rumexchangelog.Warningf("receive a non SYNC type package, %s", pkg.Type)
	}

	return fmt.Errorf("unsupported package type: %s", pkg.Type)
}
