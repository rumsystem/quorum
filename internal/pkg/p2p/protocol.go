package p2p

import (
	"github.com/golang/glog"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	msgio "github.com/libp2p/go-msgio"
	"google.golang.org/protobuf/proto"
)

const HeadBlockProtocolID = "/quorum/headblocks/1.0.0"

type HeadBlockService struct {
	Host       host.Host
	Relationdb interface{}
}

func NewHeadBlockService(h host.Host, relationdb interface{}) *HeadBlockService {
	ps := &HeadBlockService{Host: h, Relationdb: relationdb}
	h.SetStreamHandler(HeadBlockProtocolID, ps.HeadBlockHandler)
	return ps
}

func (service *HeadBlockService) HeadBlockHandler(s network.Stream) {
	reader := msgio.NewVarintReaderSize(s, network.MessageSizeMax)
	for {
		msg, err := reader.ReadMsg()
		if err == nil {
			pb := &quorumpb.BlockMessage{}
			err = proto.Unmarshal(msg, pb)
			reader.ReleaseMsg(msg)
			glog.Infof("receive message: %s from %s\n", pb, s.Conn().RemotePeer())

			var blockmsg *quorumpb.BlockMessage
			switch msgType := pb.Type; msgType {
			case quorumpb.BlockMessage_ASKHEAD:
				blockmsg = &quorumpb.BlockMessage{
					Type:  quorumpb.BlockMessage_REPLYHEAD,
					Value: "000099",
				}
			case quorumpb.BlockMessage_ASKNEXT:
				blockmsg = &quorumpb.BlockMessage{
					Type:  quorumpb.BlockMessage_REPLYNEXT,
					Value: "00005",
				}
			}
			if blockmsg != nil {
				replymsg, err := proto.Marshal(blockmsg)
				mw := msgio.NewWriter(s)
				err = mw.WriteMsg(replymsg)
				if err != nil {
					glog.Errorf("HeadBlockHandler WriteMsg error: %s", err)
				}
			}
			s.Close()
			return
		} else {
			glog.Errorf("HeadBlockHandler ReadMsg error: %s", err)
			s.Reset()
		}
	}
}
