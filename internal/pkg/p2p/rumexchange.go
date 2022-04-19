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
	"github.com/rumsystem/quorum/internal/pkg/stats"
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
	streampool         sync.Map //map[peer.ID]streamPoolItem
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

type streamPoolItem struct {
	s      network.Stream
	cancel context.CancelFunc
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

func (r *RexService) GetStream(peerid peer.ID) (*streamPoolItem, error) {

	poolitem, ok := r.streampool.Load(peerid)
	if ok {
		return poolitem.(*streamPoolItem), nil
	}
	//new stream
	ctx, cancel := context.WithCancel(context.Background())
	s, err := r.Host.NewStream(ctx, peerid, r.ProtocolId)
	newpoolitem := &streamPoolItem{s: s, cancel: cancel}
	if err != nil {
		return nil, err
	}

	go r.HandlerProcessloop(ctx, s)
	r.streampool.Store(peerid, newpoolitem)
	return newpoolitem, nil
}

func (r *RexService) ChainReg(groupid string, cdhIface iface.ChainDataHandlerIface) {
	_, ok := r.chainmgr[groupid]
	if ok == false {
		r.chainmgr[groupid] = cdhIface
		rumexchangelog.Debugf("chain reg with rumexchange: %s", groupid)
	}
}

func (r *RexService) PublishToStream(msg *quorumpb.RumMsg, s network.Stream) error {
	remotePeer := s.Conn().RemotePeer()
	log := stats.NetworkStats{
		From:      r.Host.ID(),
		To:        remotePeer,
		Action:    stats.PublishToStream,
		Direction: network.DirOutbound,
		Size:      stats.GetProtoSize(msg),
	}
	rumexchangelog.Debugf("PublishResponse msg to peer: %s", remotePeer)
	bufw := bufio.NewWriter(s)
	wc := protoio.NewDelimitedWriter(bufw)
	err := wc.WriteMsg(msg)
	if err != nil {
		rumexchangelog.Debugf("writemsg to network stream err: %s", err)

		log.Success = false
		if err := stats.GetStatsDB().AddNetworkLog(&log); err != nil {
			rumexchangelog.Warningf("add network log to db failed: %s", err)
		}

		return err
	} else {
		rumexchangelog.Debugf("writemsg to network stream succ: %s.", remotePeer)

		log.Success = true
		if err := stats.GetStatsDB().AddNetworkLog(&log); err != nil {
			rumexchangelog.Warningf("add network log to db failed: %s", err)
		}
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

	poolitem, err := r.GetStream(toid)
	if err != nil {
		rumexchangelog.Debugf("create network stream to %s err: %s", to, err)
		return err
	}
	s := poolitem.s
	remotePeer := s.Conn().RemotePeer()

	log := stats.NetworkStats{
		From:      r.Host.ID(),
		To:        remotePeer,
		Action:    stats.PublishToPeerID,
		Direction: network.DirOutbound,
		Size:      stats.GetProtoSize(msg),
	}

	bufw := bufio.NewWriter(s)
	wc := protoio.NewDelimitedWriter(bufw)
	err = wc.WriteMsg(msg)
	if err != nil {
		rumexchangelog.Debugf("writemsg to network stream err: %s", err)
		r.streampool.Delete(remotePeer)
		r.peerstore.AddIgnorePeer(toid)
		s.Close()

		log.Success = false
		if err := stats.GetStatsDB().AddNetworkLog(&log); err != nil {
			rumexchangelog.Warningf("add network log to db failed: %s", err)
		}

		return err
	} else {
		log.Success = true
		if err := stats.GetStatsDB().AddNetworkLog(&log); err != nil {
			rumexchangelog.Warningf("add network log to db failed: %s", err)
		}

		rumexchangelog.Debugf("writemsg to network stream succ: %s.", to)
	}
	bufw.Flush()

	return nil
}

//Publish to 5 random connected peers
func (r *RexService) Publish(groupid string, msg *quorumpb.RumMsg) error {
	//TODO: select peers
	succ := 0
	peers := r.Host.Network().Peers()
	maxnum := 5

	randompeerlist := r.peerstore.GetRandomPeer(groupid, maxnum, peers)

	for _, p := range randompeerlist {
		if err := r.PublishToPeerId(msg, peer.Encode(p)); err != nil {
			rumexchangelog.Debugf("writemsg to network stream err: %s", err)
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
	if err != nil {
		return err
	}

	if err := r.PublishToPeerId(msg, peer.Encode(p)); err != nil {
		rumexchangelog.Debugf("writemsg to network stream err: %s", err)
		return err
	}
	rumexchangelog.Debugf("writemsg to network stream succ: %s. wait the response", p)
	return nil
}

func (r *RexService) PrivateChannelReady(connrespmsg *quorumpb.SessionConnResp) {
	noti := RexNotification{JoinChannel, connrespmsg.ChannelId}
	r.notificationch <- noti
	rumexchangelog.Debugf("join channel %s notification emit %s.", connrespmsg.ChannelId, r.Host.ID())
}

func (r *RexService) HandleRumExchangeMsg(rummsg *quorumpb.RumMsg, s network.Stream) {
	remotePeer := s.Conn().RemotePeer()
	localPeer := r.Host.ID()
	log := stats.NetworkStats{
		From:      localPeer,
		To:        remotePeer,
		Direction: network.DirInbound,
		Size:      stats.GetProtoSize(rummsg),
		Success:   true,
	}
	log.Action = log.Action.GetByRumMsgType(rummsg.MsgType)

	switch rummsg.MsgType {
	case quorumpb.RumMsgType_RELAY_REQ, quorumpb.RumMsgType_RELAY_RESP:
		for _, v := range r.msgtypehandlers {
			if v.Name == "rumrelay" {
				if err := stats.GetStatsDB().AddNetworkLog(&log); err != nil {
					networklog.Warningf("add network log to db failed: %s", err)
				}

				v.Handler(rummsg, s)
				break
			}
		}
	case quorumpb.RumMsgType_IF_CONN, quorumpb.RumMsgType_CONN_RESP:
		for _, v := range r.msgtypehandlers {
			if v.Name == "rumsession" {
				if err := stats.GetStatsDB().AddNetworkLog(&log); err != nil {
					networklog.Warningf("add network log to db failed: %s", err)
				}

				v.Handler(rummsg, s)
				break
			}
		}
	case quorumpb.RumMsgType_CHAIN_DATA:
		for _, v := range r.msgtypehandlers {
			if v.Name == "rumchaindata" {
				if err := stats.GetStatsDB().AddNetworkLog(&log); err != nil {
					networklog.Warningf("add network log to db failed: %s", err)
				}

				v.Handler(rummsg, s)
				break
			}
		}
	}
}

func (r *RexService) Handler(s network.Stream) {
	ctx := context.Background()
	r.HandlerProcessloop(ctx, s)
}

func (r *RexService) HandlerProcessloop(ctx context.Context, s network.Stream) {
	remotePeer := s.Conn().RemotePeer()

	reader := msgio.NewVarintReaderSize(s, network.MessageSizeMax)
	defer rumexchangelog.Debugf("RumExchange stream handler %s exit", remotePeer)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msgdata, err := reader.ReadMsg()
			if err != nil {
				if err != io.EOF {
					rumexchangelog.Debugf("RumExchange stream handler from %s error: %s stream reset", s.Conn().RemotePeer(), err)
					_ = s.Reset()
				} else {
					rumexchangelog.Debugf("RumExchange stream handler EOF %s", remotePeer)
					r.streampool.Delete(remotePeer)
					_ = s.Close()
				}
				return
			}

			var rummsg quorumpb.RumMsg
			if err = proto.Unmarshal(msgdata, &rummsg); err == nil {
				r.HandleRumExchangeMsg(&rummsg, s)
			}
		}
	}

}
func (r *RexService) handlePackage(pkg *quorumpb.Package, s network.Stream) {
	if pkg.Type == quorumpb.PackageType_TRX {
		rumexchangelog.Debugf("receive a trx, from %s", s.Conn().RemotePeer())
		var trx *quorumpb.Trx
		trx = &quorumpb.Trx{}
		err := proto.Unmarshal(pkg.Data, trx)
		if err == nil {
			chainDataHandler, ok := r.chainmgr[trx.GroupId]
			if ok == true {
				r.peerstore.Save(trx.GroupId, s.Conn().RemotePeer(), peerstoreTTL)
				chainDataHandler.HandleTrxRex(trx, s)
			} else {
				rumexchangelog.Debugf("receive a group unknown package, groupid: %s from: %s", trx.GroupId, s.Conn().RemotePeer())
			}
		} else {
			rumexchangelog.Debugf(err.Error())
		}
	} else {
		rumexchangelog.Warningf("receive a non-trx package, %s", pkg.Type)
	}
}

type netNotifiee RexService

func (nn *netNotifiee) RexService() *RexService {
	return (*RexService)(nn)
}

func (nn *netNotifiee) Connected(n network.Network, v network.Conn)      {}
func (nn *netNotifiee) Disconnected(n network.Network, v network.Conn)   {}
func (nn *netNotifiee) OpenedStream(n network.Network, s network.Stream) {}
func (nn *netNotifiee) ClosedStream(n network.Network, v network.Stream) {}
func (nn *netNotifiee) Listen(n network.Network, a ma.Multiaddr)         {}
func (nn *netNotifiee) ListenClose(n network.Network, a ma.Multiaddr)    {}
