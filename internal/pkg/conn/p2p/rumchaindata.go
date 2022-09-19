package p2p

import (
	"fmt"

	"github.com/libp2p/go-libp2p-core/network"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type RexChainData struct {
	rex *RexService
}

func NewRexChainData(rex *RexService) *RexChainData {
	return &RexChainData{rex: rex}
}

func (r *RexChainData) Handler(rummsg *quorumpb.RumMsg, s network.Stream) error {
	frompeerid := s.Conn().RemotePeer()
	pkg := rummsg.DataPackage

	if pkg.Type == quorumpb.PackageType_TRX {
		rumexchangelog.Debugf("receive a trx, from %s", frompeerid)
		trx := &quorumpb.Trx{}
		err := proto.Unmarshal(pkg.Data, trx)
		if err == nil {
			targetchain, ok := r.rex.chainmgr[trx.GroupId]
			if ok == true {
				return targetchain.HandleTrxRex(trx, s)
			} else {
				rumexchangelog.Warningf("receive a group unknown package, groupid: %s from: %s", trx.GroupId, frompeerid)
			}
		} else {
			rumexchangelog.Warningf(err.Error())
		}
	} else {
		rumexchangelog.Warningf("receive a non-trx package, %s", pkg.Type)
	}

	return fmt.Errorf("unsupported trx type: %s", pkg.Type)
}
