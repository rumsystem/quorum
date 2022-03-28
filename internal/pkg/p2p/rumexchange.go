package p2p

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	msgio "github.com/libp2p/go-msgio"
	"github.com/libp2p/go-msgio/protoio"
	ma "github.com/multiformats/go-multiaddr"
	iface "github.com/rumsystem/quorum/internal/pkg/chaindataciface"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var rumexchangelog = logging.Logger("rumexchange")
var peerstoreTTL time.Duration = time.Duration(20 * time.Minute)

const IDVer = "1.0.0"

type Chain interface {
	HandleTrxWithRex(trx *quorumpb.Trx, from peer.ID) error
	HandleBlockWithRex(block *quorumpb.Block, from peer.ID) error
}

type RumHandlerFunc func(msg *quorumpb.RumMsg, s network.Stream) error

type RumHandler struct {
	Handler RumHandlerFunc
	Name    string
}

type RexService struct {
	Host               host.Host
	peerStatus         *PeerStatus
	ProtocolId         protocol.ID
	notificationch     chan RexNotification
	chainmgr           map[string]iface.ChainDataHandlerIface
	peerstore          *RumGroupPeerStore
	msgtypehandlers    []RumHandler
	streampool         sync.Map //map[peer.ID]network.Stream
	msgtypehandlerlock sync.RWMutex
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
	chainmgr := make(map[string]iface.ChainDataHandlerIface)
	rumpeerstore := &RumGroupPeerStore{}
	rexs := &RexService{Host: h, peerStatus: peerStatus, peerstore: rumpeerstore, ProtocolId: protocol.ID(customprotocol), notificationch: notification, chainmgr: chainmgr}
	rumexchangelog.Debug("new rex service")
	h.SetStreamHandler(rexs.ProtocolId, rexs.Handler)
	rumexchangelog.Debugf("new rex service SetStreamHandler: %s", customprotocol)
	return rexs
}

func (r *RexService) SetDelegate() {
	r.Host.Network().Notify((*netNotifiee)(r))
}

func (r *RexService) SetHandlerMatchMsgType(name string, handler RumHandlerFunc) {

	r.msgtypehandlerlock.Lock()
	defer r.msgtypehandlerlock.Unlock()
	for i, v := range r.msgtypehandlers {
		if v.Name == name {
			r.msgtypehandlers[i] = RumHandler{handler, name}
			return
		}
	}
	r.msgtypehandlers = append(r.msgtypehandlers, RumHandler{handler, name})
}

func (r *RexService) ConnectRex(ctx context.Context) error {
	peers := r.Host.Network().Peers()
	rumexchangelog.Debugf("try (%d) peers.", len(peers))
	for _, p := range peers {
		if r.peerStatus.IfSkip(p, r.ProtocolId) == false {
			s, err := r.Host.NewStream(ctx, p, r.ProtocolId)
			if err != nil {
				rumexchangelog.Debugf("create network stream err: %s", err)
				r.peerStatus.Update(p, r.ProtocolId, PROTOCOL_NOT_SUPPORT)
			} else {
				rumexchangelog.Debugf("create network stream success %s.", p)
				s.Close()
			}
		}
	}
	return nil
}

func (r *RexService) GetStream(peerid peer.ID) (network.Stream, error) {
	stream, ok := r.streampool.Load(peerid)
	if ok {
		return stream.(network.Stream), nil
	}
	//new stream
	ctx := context.Background()
	s, err := r.Host.NewStream(ctx, peerid, r.ProtocolId)
	if err == nil {
		r.streampool.Store(peerid, s)
	}
	return s, err
}

func (r *RexService) ChainReg(groupid string, cdhIface iface.ChainDataHandlerIface) {
	rumexchangelog.Debugf("disabled call chain reg : %s", groupid)
	_, ok := r.chainmgr[groupid]
	if ok == false {
		r.chainmgr[groupid] = cdhIface
		rumexchangelog.Debugf("chain reg with rumexchange: %s", groupid)
	}
}

func (r *RexService) PublishToStream(msg *quorumpb.RumMsg, s network.Stream) error {
	rumexchangelog.Debugf("PublishResponse msg to peer: %s", s.Conn().RemotePeer())
	bufw := bufio.NewWriter(s)
	wc := protoio.NewDelimitedWriter(bufw)
	err := wc.WriteMsg(msg)
	if err != nil {
		rumexchangelog.Debugf("writemsg to network stream err: %s", err)
		return err
	} else {
		rumexchangelog.Debugf("writemsg to network stream succ: %s.", s.Conn().RemotePeer())
	}
	bufw.Flush()
	return nil
}

func (r *RexService) PublishToPeerId(msg *quorumpb.RumMsg, to string) error {
	rumexchangelog.Debugf("PublishResponse msg to peer: %s", to)
	toid, err := peer.Decode(to)
	if err != nil {
		return err
	}

	//s, err := r.Host.NewStream(ctx, toid, r.ProtocolId)
	s, err := r.GetStream(toid)
	if err != nil {
		rumexchangelog.Debugf("create network stream to %s err: %s", to, err)
		return err
	}
	//defer func() { _ = s.Close() }()
	bufw := bufio.NewWriter(s)
	wc := protoio.NewDelimitedWriter(bufw)
	err = wc.WriteMsg(msg)
	if err != nil {
		rumexchangelog.Debugf("writemsg to network stream err: %s", err)
		r.streampool.Delete(s.Conn().RemotePeer())
		s.Close()
		return err
	} else {
		rumexchangelog.Debugf("writemsg to network stream succ: %s.", to)
	}
	bufw.Flush()
	return nil
}

//Publish to All connected peers
func (r *RexService) Publish(groupid string, msg *quorumpb.RumMsg) error {
	//TODO: select peers
	succ := 0
	peers := r.Host.Network().Peers()
	maxnum := 5

	randompeerlist := r.peerstore.GetRandomPeer(groupid, maxnum, peers)

	for _, p := range randompeerlist {
		//ctx := context.Background()
		//s, err := r.Host.NewStream(ctx, p, r.ProtocolId)
		s, err := r.GetStream(p)
		if err != nil {
			rumexchangelog.Debugf("create network stream err: %s", err)
			r.peerstore.AddIgnorePeer(p)
			continue
		}
		bufw := bufio.NewWriter(s)
		wc := protoio.NewDelimitedWriter(bufw)
		err = wc.WriteMsg(msg)
		bufw.Flush()
		if err != nil {
			rumexchangelog.Debugf("writemsg to network stream err: %s", err)
			r.streampool.Delete(s.Conn().RemotePeer())
			s.Close()
		} else {
			succ++
			rumexchangelog.Debugf("writemsg to network stream succ: %s.", p)
		}

	}

	return nil
}

//Publish to one random peer
func (r *RexService) PublishToOneRandom(msg *quorumpb.RumMsg) error {
	rumexchangelog.Debugf("PublishToOneRandom called")
	peers := r.Host.Network().Peers()
	p, err := r.peerstore.GetOneRandomPeer(peers)
	rumexchangelog.Debugf("PublishToOneRandom to peer: %s err:", p, err)
	if err == nil {
		s, err := r.GetStream(p)
		if err != nil {
			rumexchangelog.Debugf("create network stream err: %s", err)
			r.peerstore.AddIgnorePeer(p)
			return err
		}
		bufw := bufio.NewWriter(s)
		wc := protoio.NewDelimitedWriter(bufw)
		err = wc.WriteMsg(msg)
		bufw.Flush()
		if err != nil {
			rumexchangelog.Debugf("writemsg to network stream err: %s", err)
		} else {
			rumexchangelog.Debugf("writemsg to network stream succ: %s. wait the response", p)
		}

	}
	return err
}

func (r *RexService) PrivateChannelReady(connrespmsg *quorumpb.SessionConnResp) {
	noti := RexNotification{JoinChannel, connrespmsg.ChannelId}
	r.notificationch <- noti
	rumexchangelog.Debugf("join channel %s notification emit %s.", connrespmsg.ChannelId, r.Host.ID())
}

func (r *RexService) HandleRumExchangeMsg(rummsg *quorumpb.RumMsg, s network.Stream) {
	switch rummsg.MsgType {
	case quorumpb.RumMsgType_RELAY_REQ, quorumpb.RumMsgType_RELAY_RESP:
		for _, v := range r.msgtypehandlers {
			if v.Name == "rumrelay" {
				v.Handler(rummsg, s)
				break
			}
		}
	case quorumpb.RumMsgType_IF_CONN, quorumpb.RumMsgType_CONN_RESP:
		for _, v := range r.msgtypehandlers {
			if v.Name == "rumsession" {
				v.Handler(rummsg, s)
				break
			}
		}
	case quorumpb.RumMsgType_CHAIN_DATA:
		rumexchangelog.Debugf("type is CHAIN_DATA")
		for _, v := range r.msgtypehandlers {
			if v.Name == "rumchaindata" {
				v.Handler(rummsg, s)
				break
			}
		}
	}
}

func (r *RexService) Handler(s network.Stream) {
	reader := msgio.NewVarintReaderSize(s, network.MessageSizeMax)
	rumexchangelog.Debugf("RumExchange stream handler start")
	for {
		msgdata, err := reader.ReadMsg()
		if err != nil {
			if err != io.EOF {
				rumexchangelog.Warningf("RumExchange stream handler from %s error: %s stream reset", s.Conn().RemotePeer(), err)
				r.streampool.Delete(s.Conn().RemotePeer())
				_ = s.Close()
			} else {
				rumexchangelog.Warningf("RumExchange stream handler EOF %s", s.Conn().RemotePeer())
				_ = s.Reset()
			}
			return
		}

		var rummsg quorumpb.RumMsg
		err = proto.Unmarshal(msgdata, &rummsg)
		if err == nil {
			r.HandleRumExchangeMsg(&rummsg, s)
		} else {
			rumexchangelog.Warningf("msg err: %s", err)
		}
	}
}

func (r *RexService) handlePackage(pkg *quorumpb.Package, s network.Stream) {
	if pkg.Type == quorumpb.PackageType_TRX {
		rumexchangelog.Infof("receive a trx, from %s", s.Conn().RemotePeer())
		var trx *quorumpb.Trx
		trx = &quorumpb.Trx{}
		err := proto.Unmarshal(pkg.Data, trx)
		if err == nil {
			chainDataHandler, ok := r.chainmgr[trx.GroupId]
			if ok == true {
				r.peerstore.Save(trx.GroupId, s.Conn().RemotePeer(), peerstoreTTL)
				chainDataHandler.HandleTrxRex(trx, s)
			} else {
				rumexchangelog.Warningf("receive a group unknown package, groupid: %s from: %s", trx.GroupId, s.Conn().RemotePeer())
			}
		} else {
			rumexchangelog.Warningf(err.Error())
		}
	} else {
		rumexchangelog.Warningf("receive a non-trx package, %s", pkg.Type)
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
