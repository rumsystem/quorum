package p2p

import (
	"github.com/golang/glog"
    "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	msgio "github.com/libp2p/go-msgio"
)

const HeadBlockProtocolID = "/quorum/headblocks/1.0.0"

type HeadBlockService struct {
	Host host.Host
    Relationdb interface{}
}

func NewHeadBlockService(h host.Host , relationdb interface{} ) *HeadBlockService {
    ps := &HeadBlockService{Host:h, Relationdb: relationdb}
	h.SetStreamHandler(HeadBlockProtocolID, ps.HeadBlockHandler)
	return ps
}

func (service *HeadBlockService) HeadBlockHandler(s network.Stream) {
	reader := msgio.NewVarintReaderSize(s, network.MessageSizeMax)
	for {
		msg, err := reader.ReadMsg()
		if len(msg)>0 {
			if err != nil {
                glog.Errorf("HeadBlockHandler ReadMsg error: %s",err)
				s.Reset()
			}else {
				glog.Infof("receive message: %s\n", msg)
                newmsg := []byte("HEAD:00001")
				mw := msgio.NewWriter(s)
				err := mw.WriteMsg(newmsg)
                if err != nil{
                    glog.Errorf("HeadBlockHandler WriteMsg error: %s",err)
                }
				s.Close()
			}
			return
		}
	}
}
