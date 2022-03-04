package p2p

import (
	"fmt"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

type RexSession struct {
	rex *RexService
}

func NewRexSession(rex *RexService) *RexSession {
	return &RexSession{rex: rex}
}

func (r *RexSession) InitSession(peerid string, channelid string) error {
	privateid, err := peer.Decode(peerid)
	if err != nil {
		rumexchangelog.Warningf("decode perrid err: %s", err)
	}
	ifconnmsg := &quorumpb.SessionIfConn{DestPeerID: []byte(privateid), SrcPeerID: []byte(r.rex.Host.ID()), ChannelId: channelid}
	sessionmsg := &quorumpb.RumMsg{MsgType: quorumpb.RumMsgType_IF_CONN, IfConn: ifconnmsg}

	succ := 0

	peers := r.rex.Host.Network().Peers()
	for _, p := range peers {
		if r.rex.peerStatus.IfSkip(p, r.rex.ProtocolId) == false {
			err := r.rex.PublishTo(sessionmsg, p)
			if err != nil {
				rumexchangelog.Warningf("writemsg to network stream err: %s", err)
			} else {
				succ++
			}
		}

	}
	if succ > 0 {
		return nil
	} else {
		return fmt.Errorf("no enough peer to send msg")
	}
}

func (r *RexSession) DestPeerResp(recvfrom peer.ID, ifconnmsg *quorumpb.SessionIfConn) {

	connrespmsg := &quorumpb.SessionConnResp{DestPeerID: ifconnmsg.SrcPeerID, SrcPeerID: ifconnmsg.DestPeerID, SessionToken: ifconnmsg.SessionToken, Peersroutes: ifconnmsg.Peersroutes, ChannelId: ifconnmsg.ChannelId}

	noti := RexNotification{JoinChannel, connrespmsg.ChannelId}
	r.rex.notificationch <- noti
	rumexchangelog.Debugf("join channel %s notification emit %s.", connrespmsg.ChannelId, r.rex.Host.ID())

	sessionmsg := &quorumpb.RumMsg{MsgType: quorumpb.RumMsgType_CONN_RESP, ConnResp: connrespmsg}

	err := r.rex.PublishTo(sessionmsg, recvfrom)
	if err != nil {
		rumexchangelog.Debugf("msg.Peersroutes num:(%d) resp success.", len(sessionmsg.ConnResp.Peersroutes))
	} else {
		rumexchangelog.Debugf("Write connresp back to %s , err %s", recvfrom, err)
	}
}

func (r *RexSession) PassIfConnMsgToNext(recvfrom peer.ID, ifconnmsg *quorumpb.SessionIfConn) error {
	peersig := &quorumpb.PeerSig{PeerId: []byte(r.rex.Host.ID())}
	peers := r.rex.Host.Network().Peers()
	if len(ifconnmsg.Peersroutes) >= 3 {
		return fmt.Errorf("reatch max msg pass level: %d", len(ifconnmsg.Peersroutes))
	}
	ifconnmsg.Peersroutes = append(ifconnmsg.Peersroutes, peersig)

	rumexchangelog.Debugf("stream routes append peerid: %s", r.rex.Host.ID())

	sessionmsg := &quorumpb.RumMsg{MsgType: quorumpb.RumMsgType_IF_CONN, IfConn: ifconnmsg}
	succ := 0

	for _, p := range peers {
		if succ >= 5 {
			rumexchangelog.Debugf("max rex publish peers (%d) reached, pause. ", succ)
			break
		}
		if p != r.rex.Host.ID() && p != peer.ID(sessionmsg.IfConn.SrcPeerID) && p != recvfrom && r.rex.peerStatus.IfSkip(p, r.rex.ProtocolId) == false { //not myself, not src peer, not recvfrom this peer, not be skip so passnext
			err := r.rex.PublishTo(sessionmsg, p)
			if err != nil {
				rumexchangelog.Debugf("PassIfConnMsgToNext network stream err: %s on %s", err, p)
			} else {
				rumexchangelog.Debugf("writemsg to network stream succ.")
				succ++
			}
			rumexchangelog.Debugf("msg.Peersroutes: (%d)", len(sessionmsg.IfConn.Peersroutes))
		}
	}

	if succ > 0 {
		return nil
	} else {
		return fmt.Errorf("no enough peer to send msg")
	}
}

func (r *RexSession) PassConnRespMsgToNext(connrespmsg *quorumpb.SessionConnResp) {
	//find the next peer to pass
	var nextpeerid peer.ID
	peers := r.rex.Host.Network().Peers()
	for idx, p := range connrespmsg.Peersroutes {
		pid, err := peer.IDFromBytes(p.PeerId)
		if err == nil && pid == r.rex.Host.ID() {
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
				if r.rex.peerStatus.IfSkip(cp, r.rex.ProtocolId) == true {
					continue
				}

				sessionmsg := &quorumpb.RumMsg{MsgType: quorumpb.RumMsgType_CONN_RESP, ConnResp: connrespmsg}
				err := r.rex.PublishTo(sessionmsg, nextpeerid)
				if err != nil {
					rumexchangelog.Debugf("PassConnRespMsgToNext network stream err: %s on %s", err, cp)
				} else {
					noti := RexNotification{JoinChannel, connrespmsg.ChannelId}
					r.rex.notificationch <- noti
					rumexchangelog.Debugf("join channel %s notification emit %s.", connrespmsg.ChannelId, r.rex.Host.ID())
					rumexchangelog.Debugf("pass respmsg to %s, write err %s", nextpeerid, err)
					rumexchangelog.Debugf("msg.Peersroutes: %s", sessionmsg.ConnResp.Peersroutes)
				}
				break
			}
		}
	}

}

func (r *RexSession) Handler(rummsg *quorumpb.RumMsg, s network.Stream) {
	switch rummsg.MsgType {
	case quorumpb.RumMsgType_IF_CONN:
		rumexchangelog.Debugf("type is SessionIfConn")
		if peer.ID(rummsg.IfConn.DestPeerID) == r.rex.Host.ID() {
			rumexchangelog.Debugf("msg.Peersroutes: %s", rummsg.IfConn.Peersroutes)
			rumexchangelog.Debugf("the dest peer is me, join the channel and response.")
			r.DestPeerResp(s.Conn().RemotePeer(), rummsg.IfConn)
		} else if peer.ID(rummsg.IfConn.SrcPeerID) == r.rex.Host.ID() {
			rumexchangelog.Debugf("the src peer is me, skip")
		} else {
			r.PassIfConnMsgToNext(s.Conn().RemotePeer(), rummsg.IfConn)
			//ok passto next
		}
	case quorumpb.RumMsgType_CONN_RESP:
		rumexchangelog.Debugf("type is SessionConnResp")
		if peer.ID(rummsg.ConnResp.DestPeerID) == r.rex.Host.ID() {
			rumexchangelog.Debugf("msg.Peersroutes:%s", rummsg.ConnResp.Peersroutes)
			rumexchangelog.Debugf("the dest peer is me, the private channel should be ready.")
			//r.PrivateChannelReady(sessionmsg.ConnResp) //FOR TEST

		} else if peer.ID(rummsg.ConnResp.SrcPeerID) == r.rex.Host.ID() {
			rumexchangelog.Debugf("the src peer is me, skip")
		} else {
			r.PassConnRespMsgToNext(rummsg.ConnResp)
		}
	}
}
