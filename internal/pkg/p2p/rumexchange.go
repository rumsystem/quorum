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
	ma "github.com/multiformats/go-multiaddr"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
	"io"
)

var rumexchangelog = logging.Logger("rumexchange")

const IDVer = "1.0.0"

type RexService struct {
	Host           host.Host
	peerStatus     *PeerStatus
	ProtocolId     protocol.ID
	notificationch chan RexNotification
}

type ActionType int

const (
	JoinChannel ActionType = iota
	JoinChannelAndPublishTest
)

type RexNotification struct {
	Action    ActionType
	ChannelId string
}

func NewRexService(h host.Host, peerStatus *PeerStatus, Networkname string, ProtocolPrefix string, notification chan RexNotification) *RexService {
	customprotocol := fmt.Sprintf("%s/%s/rex/%s", ProtocolPrefix, Networkname, IDVer)
	rexs := &RexService{h, peerStatus, protocol.ID(customprotocol), notification}
	rumexchangelog.Debug("new rex service")
	h.SetStreamHandler(rexs.ProtocolId, rexs.Handler)
	rumexchangelog.Debugf("new rex service SetStreamHandler: %s", customprotocol)
	return rexs
}

func (r *RexService) SetDelegate() {
	r.Host.Network().Notify((*netNotifiee)(r))
}

func (r *RexService) ConnectRex(ctx context.Context) error {
	peers := r.Host.Network().Peers()
	rumexchangelog.Debugf("try (%d) peers.", len(peers))
	for _, p := range peers {
		if r.peerStatus.IfSkip(p, r.ProtocolId) == false {
			_, err := r.Host.NewStream(ctx, p, r.ProtocolId)
			if err != nil {
				rumexchangelog.Debugf("create network stream err: %s", err)
				r.peerStatus.Update(p, r.ProtocolId, PROTOCOL_NOT_SUPPORT)
			} else {
				rumexchangelog.Debugf("create network stream success.")
			}
		}
	}
	return nil
}

func (r *RexService) InitSession(peerid string, channelid string) error {
	privateid, err := peer.Decode(peerid)
	if err != nil {
		rumexchangelog.Warningf("decode perrid err: %s", err)
	}
	ifconnmsg := &quorumpb.SessionIfConn{DestPeerID: []byte(privateid), SrcPeerID: []byte(r.Host.ID()), ChannelId: channelid}
	sessionmsg := &quorumpb.RumMsg{MsgType: quorumpb.RumMsgType_IF_CONN, IfConn: ifconnmsg}

	succ := 0

	peers := r.Host.Network().Peers()
	for _, p := range peers {
		if r.peerStatus.IfSkip(p, r.ProtocolId) == false {
			ctx := context.Background()
			s, err := r.Host.NewStream(ctx, p, r.ProtocolId)
			if err != nil {
				rumexchangelog.Debugf("create network stream err: %s", err)
				r.peerStatus.Update(p, r.ProtocolId, PROTOCOL_NOT_SUPPORT)
			} else {
				bufw := bufio.NewWriter(s)
				wc := protoio.NewDelimitedWriter(bufw)
				err := wc.WriteMsg(sessionmsg)
				if err != nil {
					rumexchangelog.Warningf("writemsg to network stream err: %s", err)
				} else {
					succ++
					rumexchangelog.Debugf("writemsg to network stream succ: %s.", p)
				}
				bufw.Flush()
			}
		}

	}
	if succ > 0 {
		return nil
	} else {
		return fmt.Errorf("no enough peer to send msg")
	}
}

func (r *RexService) DestPeerResp(recvfrom peer.ID, ifconnmsg *quorumpb.SessionIfConn) {

	connrespmsg := &quorumpb.SessionConnResp{DestPeerID: ifconnmsg.SrcPeerID, SrcPeerID: ifconnmsg.DestPeerID, SessionToken: ifconnmsg.SessionToken, Peersroutes: ifconnmsg.Peersroutes, ChannelId: ifconnmsg.ChannelId}

	noti := RexNotification{JoinChannel, connrespmsg.ChannelId}
	r.notificationch <- noti
	rumexchangelog.Debugf("join channel %s notification emit %s.", connrespmsg.ChannelId, r.Host.ID())

	sessionmsg := &quorumpb.RumMsg{MsgType: quorumpb.RumMsgType_CONN_RESP, ConnResp: connrespmsg}
	ctx := context.Background()

	var s network.Stream
	var err error
	s, err = r.Host.NewStream(ctx, recvfrom, r.ProtocolId)
	bufw := bufio.NewWriter(s)
	wc := protoio.NewDelimitedWriter(bufw)
	err = wc.WriteMsg(sessionmsg)
	rumexchangelog.Debugf("Write connresp back to %s , err %s", s.Conn().RemotePeer(), err)
	rumexchangelog.Debugf("msg.Peersroutes:%s", sessionmsg.ConnResp.Peersroutes)
	bufw.Flush()
}

func (r *RexService) PrivateChannelReady(connrespmsg *quorumpb.SessionConnResp) {
	noti := RexNotification{JoinChannel, connrespmsg.ChannelId}
	r.notificationch <- noti
	rumexchangelog.Debugf("join channel %s notification emit %s.", connrespmsg.ChannelId, r.Host.ID())
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
				if r.peerStatus.IfSkip(cp, r.ProtocolId) == true {
					continue
				}
				ctx := context.Background()
				var s network.Stream
				var err error
				s, err = r.Host.NewStream(ctx, nextpeerid, r.ProtocolId)
				if err != nil {
					rumexchangelog.Debugf("PassConnRespMsgToNext network stream err: %s on %s", err, cp)
				} else {
					noti := RexNotification{JoinChannel, connrespmsg.ChannelId}
					r.notificationch <- noti
					rumexchangelog.Debugf("join channel %s notification emit %s.", connrespmsg.ChannelId, r.Host.ID())

					bufw := bufio.NewWriter(s)
					wc := protoio.NewDelimitedWriter(bufw)
					sessionmsg := &quorumpb.RumMsg{MsgType: quorumpb.RumMsgType_CONN_RESP, ConnResp: connrespmsg}
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

func (r *RexService) PassIfConnMsgToNext(recvfrom peer.ID, ifconnmsg *quorumpb.SessionIfConn) error {
	peersig := &quorumpb.PeerSig{PeerId: []byte(r.Host.ID())}
	peers := r.Host.Network().Peers()
	if len(ifconnmsg.Peersroutes) >= 3 {
		return fmt.Errorf("reatch max msg pass level: %d", len(ifconnmsg.Peersroutes))
	}
	ifconnmsg.Peersroutes = append(ifconnmsg.Peersroutes, peersig)

	rumexchangelog.Debugf("stream routes append peerid: %s", r.Host.ID())

	sessionmsg := &quorumpb.RumMsg{MsgType: quorumpb.RumMsgType_IF_CONN, IfConn: ifconnmsg}
	succ := 0

	ctx := context.Background()
	for _, p := range peers {
		if succ >= 5 {
			rumexchangelog.Debugf("max rex publish peers (%d) reached, pause. ", succ)
			break
		}
		if p != r.Host.ID() && p != peer.ID(sessionmsg.IfConn.SrcPeerID) && p != recvfrom && r.peerStatus.IfSkip(p, r.ProtocolId) == false { //not myself, not src peer, not recvfrom this peer, not be skip so passnext
			var s network.Stream
			var err error
			s, err = r.Host.NewStream(ctx, p, r.ProtocolId)
			if err != nil {
				rumexchangelog.Debugf("PassIfConnMsgToNext network stream err: %s on %s", err, p)
			} else {
				bufw := bufio.NewWriter(s)
				wc := protoio.NewDelimitedWriter(bufw)
				err := wc.WriteMsg(sessionmsg)

				if err != nil {
					rumexchangelog.Warningf("writemsg to network stream err: %s", err)
				} else {
					succ++
					rumexchangelog.Debugf("writemsg to network stream succ.")
				}

				rumexchangelog.Debugf("write to %s, err %s", p, err)
				rumexchangelog.Debugf("msg.Peersroutes: %s", sessionmsg.IfConn.Peersroutes)
				bufw.Flush()
			}
		}
	}

	if succ > 0 {
		return nil
	} else {
		return fmt.Errorf("no enough peer to send msg")
	}
}

func (r *RexService) Handler(s network.Stream) {
	reader := msgio.NewVarintReaderSize(s, network.MessageSizeMax)
	rumexchangelog.Debugf("RumExchange stream handler start")
	for {
		msgdata, err := reader.ReadMsg()
		if err != nil {
			rumexchangelog.Warningf("rum exchange read err: %s", err)
			if err != io.EOF {
				_ = s.Reset()
				s.Close()
				rumexchangelog.Warningf("RumExchange stream handler from %s error: %s stream reset", s.Conn().RemotePeer(), err)
			}
			return
		}

		var rummsg quorumpb.RumMsg
		err = proto.Unmarshal(msgdata, &rummsg)
		rumexchangelog.Debugf("rummsg: %s", rummsg)
		if err == nil {
			switch rummsg.MsgType {
			case quorumpb.RumMsgType_IF_CONN:
				rumexchangelog.Debugf("type is SessionIfConn")
				if peer.ID(rummsg.IfConn.DestPeerID) == r.Host.ID() {
					rumexchangelog.Debugf("msg.Peersroutes: %s", rummsg.IfConn.Peersroutes)
					rumexchangelog.Debugf("the dest peer is me, join the channel and response.")
					r.DestPeerResp(s.Conn().RemotePeer(), rummsg.IfConn)
				} else if peer.ID(rummsg.IfConn.SrcPeerID) == r.Host.ID() {
					rumexchangelog.Debugf("the src peer is me, skip")
				} else {
					r.PassIfConnMsgToNext(s.Conn().RemotePeer(), rummsg.IfConn)
					//ok passto next
				}
			case quorumpb.RumMsgType_CONN_RESP:
				rumexchangelog.Debugf("type is SessionConnResp")
				if peer.ID(rummsg.ConnResp.DestPeerID) == r.Host.ID() {
					rumexchangelog.Debugf("msg.Peersroutes:%s", rummsg.ConnResp.Peersroutes)
					rumexchangelog.Debugf("the dest peer is me, the private channel should be ready.")
					//r.PrivateChannelReady(sessionmsg.ConnResp) //FOR TEST

				} else if peer.ID(rummsg.ConnResp.SrcPeerID) == r.Host.ID() {
					rumexchangelog.Debugf("the src peer is me, skip")
				} else {
					r.PassConnRespMsgToNext(rummsg.ConnResp)
				}
			}

		} else {
			rumexchangelog.Warningf("msg err: %s", err)
		}

	}
}

type netNotifiee RexService

func (nn *netNotifiee) RexService() *RexService {
	return (*RexService)(nn)
}

func (nn *netNotifiee) Connected(n network.Network, v network.Conn) {
	rumexchangelog.Debugf("rex Connected: %s", v.RemotePeer())
}
func (nn *netNotifiee) Disconnected(n network.Network, v network.Conn) {
	rumexchangelog.Debugf("rex Disconnected: %s", v.RemotePeer())
}
func (nn *netNotifiee) OpenedStream(n network.Network, s network.Stream) {}
func (nn *netNotifiee) ClosedStream(n network.Network, v network.Stream) {}
func (nn *netNotifiee) Listen(n network.Network, a ma.Multiaddr)         {}
func (nn *netNotifiee) ListenClose(n network.Network, a ma.Multiaddr)    {}
