package p2p

import (
	"bufio"
	"context"
	"fmt"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	msgio "github.com/libp2p/go-msgio"
	"github.com/libp2p/go-msgio/protoio"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
	"io"
)

var rumexchangelog = logging.Logger("rumexchange")

const IDVer = "1.0.0"

type RexService struct {
	Host       host.Host
	ProtocolId protocol.ID
}

func NewRexService(h host.Host, Networkname string, ProtocolPrefix string) *RexService {
	customprotocol := fmt.Sprintf("%s/%s/rex/%s", ProtocolPrefix, Networkname, IDVer)
	rexs := &RexService{h, protocol.ID(customprotocol)}
	rumexchangelog.Debug("new rex service")
	h.SetStreamHandler(rexs.ProtocolId, rexs.Handler)
	rumexchangelog.Debug("new rex service SetStreamHandler: %s", customprotocol)
	return rexs
}

func NewRexObject(Networkname string, ProtocolPrefix string) *RexService {
	customprotocol := fmt.Sprintf("%s/%s/rex/%s", ProtocolPrefix, Networkname, IDVer)
	rexs := &RexService{ProtocolId: protocol.ID(customprotocol)}
	return rexs
}

func (r *RexService) DestPeerResp(recvfrom peer.ID, ifconnmsg *quorumpb.SessionIfConn) {

	connrespmsg := &quorumpb.SessionConnResp{DestPeerID: ifconnmsg.SrcPeerID, SrcPeerID: ifconnmsg.DestPeerID, SessionToken: ifconnmsg.SessionToken, Peersroutes: ifconnmsg.Peersroutes, ChannelId: "a_test_channel"}

	sessionmsg := &quorumpb.SessionMsg{MsgType: quorumpb.SessionMsgType_CONN_RESP, ConnResp: connrespmsg}
	ctx := context.Background()
	s, err := r.Host.NewStream(ctx, recvfrom, r.ProtocolId)

	bufw := bufio.NewWriter(s)
	wc := protoio.NewDelimitedWriter(bufw)
	err = wc.WriteMsg(sessionmsg)
	rumexchangelog.Debugf("Write connresp back to %s , err %s\n", s.Conn().RemotePeer(), err)
	rumexchangelog.Debugf("msg.Peersroutes:%s", sessionmsg.ConnResp.Peersroutes)
	bufw.Flush()
}

func (r *RexService) PassConnRespMsgToNext(connrespmsg *quorumpb.SessionConnResp) {
	//find the next peer to pass
	var nextpeerid peer.ID
	peers := r.Host.Network().Peers()
	for idx, p := range connrespmsg.Peersroutes {
		pid, err := peer.IDFromBytes(p.PeerId)
		if err == nil && pid == r.Host.ID() {
			if idx-1 > 0 { //myself can't be the first route peer
				nextp := connrespmsg.Peersroutes[idx-1]
				nextpeerid, err = peer.IDFromBytes(nextp.PeerId)
				break
			} else if idx == 0 {
				nextpeerid, err = peer.IDFromBytes(connrespmsg.DestPeerID)
				break
			}
		} else {
			//TODO:log erro wrong peerid
		}
	}
	if nextpeerid.Validate() == nil { //ok, pass message to the next peer
		for _, cp := range peers { //verify if the peer connected
			if cp == nextpeerid { //ok, connected, pass the message
				ctx := context.Background()
				s, err := r.Host.NewStream(ctx, nextpeerid, r.ProtocolId)
				if err != nil {
					fmt.Println(err)
				} else {
					bufw := bufio.NewWriter(s)
					wc := protoio.NewDelimitedWriter(bufw)
					sessionmsg := &quorumpb.SessionMsg{MsgType: quorumpb.SessionMsgType_CONN_RESP, ConnResp: connrespmsg}
					err := wc.WriteMsg(sessionmsg)
					rumexchangelog.Debugf("pass respmsg to %s, write err %s", nextpeerid, err)
					rumexchangelog.Debugf("msg.Peersroutes: %s", sessionmsg.ConnResp.Peersroutes)
					bufw.Flush()
				}
				break
			}
		}
	}

}

func (r *RexService) PassIfConnMsgToNext(recvfrom peer.ID, ifconnmsg *quorumpb.SessionIfConn) {
	//TODO: append my peerid to the msg

	peersig := &quorumpb.PeerSig{PeerId: []byte(r.Host.ID())}
	peers := r.Host.Network().Peers()
	ifconnmsg.Peersroutes = append(ifconnmsg.Peersroutes, peersig)

	sessionmsg := &quorumpb.SessionMsg{MsgType: quorumpb.SessionMsgType_IF_CONN, IfConn: ifconnmsg}

	ctx := context.Background()
	for _, p := range peers {
		if p != r.Host.ID() && p != peer.ID(sessionmsg.IfConn.SrcPeerID) && p != recvfrom { //not myself, not src peer, not recvfrom this peer, so passnext
			s, err := r.Host.NewStream(ctx, p, r.ProtocolId)
			if err != nil {
				fmt.Println(err)
			} else {
				bufw := bufio.NewWriter(s)
				wc := protoio.NewDelimitedWriter(bufw)
				err := wc.WriteMsg(sessionmsg)

				rumexchangelog.Debugf("write to %s, err %s", p, err)
				rumexchangelog.Debugf("msg.Peersroutes: %s", sessionmsg.IfConn.Peersroutes)
				bufw.Flush()
			}
		}
	}
}

func (r *RexService) Handler(s network.Stream) {
	//TODO: send message to a channel
	reader := msgio.NewVarintReaderSize(s, network.MessageSizeMax)

	for {
		msgdata, err := reader.ReadMsg()
		if err != nil {
			rumexchangelog.Errorf("read err: %s", err)
			if err != io.EOF {
				_ = s.Reset()
				rumexchangelog.Debugf("RumExchange stream handler from %s error: %s", s.Conn().RemotePeer(), err)
			}
			return
		}

		var sessionmsg quorumpb.SessionMsg
		err = proto.Unmarshal(msgdata, &sessionmsg)
		rumexchangelog.Debugf("sessionmsg: %s", sessionmsg)
		if err == nil {
			switch sessionmsg.MsgType {
			case quorumpb.SessionMsgType_IF_CONN:
				rumexchangelog.Debugf("type is SessionIfConn")
				if peer.ID(sessionmsg.IfConn.DestPeerID) == r.Host.ID() {
					rumexchangelog.Debugf("msg.Peersroutes: %s", sessionmsg.IfConn.Peersroutes)
					rumexchangelog.Debugf("the dest peer is me, response.")
					r.DestPeerResp(s.Conn().RemotePeer(), sessionmsg.IfConn)
				} else if peer.ID(sessionmsg.IfConn.SrcPeerID) == r.Host.ID() {
					rumexchangelog.Debugf("the src peer is me, skip")
				} else {
					r.PassIfConnMsgToNext(s.Conn().RemotePeer(), sessionmsg.IfConn)
					//ok passto next
				}
			case quorumpb.SessionMsgType_CONN_RESP:
				rumexchangelog.Debugf("type is SessionConnResp")
				if peer.ID(sessionmsg.ConnResp.DestPeerID) == r.Host.ID() {
					rumexchangelog.Debugf("msg.Peersroutes:%s", sessionmsg.ConnResp.Peersroutes)
					rumexchangelog.Debugf("the dest peer is me, the private channel should be ready.")
				} else if peer.ID(sessionmsg.ConnResp.SrcPeerID) == r.Host.ID() {
					rumexchangelog.Debugf("the src peer is me, skip")
				} else {
					r.PassConnRespMsgToNext(sessionmsg.ConnResp)
				}
			}

		} else {
			rumexchangelog.Errorf("msg err: %s", err)
		}

	}
}
