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
	chaindef "github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/metric"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
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
	chainmgr           map[string]chaindef.ChainDataSyncIface
	peerstore          *RumGroupPeerStore
	msgtypehandlers    []RumHandler
	streampool         sync.Map //map[peer.ID]streamPoolItem
	msgtypehandlerlock sync.RWMutex
}

type ActionType int

const (
	JoinChannel ActionType = iota
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
	chainmgr := make(map[string]chaindef.ChainDataSyncIface)
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

func (r *RexService) NewStream(peerid peer.ID) (*streamPoolItem, error) {
	//new stream
	ctx, cancel := context.WithCancel(context.Background())

	// could be a transient stream(relay)
	s, err := r.Host.NewStream(ctx, peerid, r.ProtocolId)
	newpoolitem := &streamPoolItem{s: s, cancel: cancel}
	if err != nil {
		return nil, err
	}
	r.streampool.Store(peerid, newpoolitem)

	go r.HandlerProcessloop(ctx, s)

	return newpoolitem, nil
}

func (r *RexService) GetStream(peerid peer.ID) (*streamPoolItem, error) {
	poolitem, ok := r.streampool.Load(peerid)
	if ok {
		streamitem := poolitem.(*streamPoolItem)
		return streamitem, nil
	}
	return r.NewStream(peerid)
}

func (r *RexService) ChainReg(groupid string, cdhIface chaindef.ChainDataSyncIface) {
	_, ok := r.chainmgr[groupid]
	if ok == false {
		r.chainmgr[groupid] = cdhIface
		rumexchangelog.Debugf("chain reg with rumexchange: %s", groupid)
	}
}

func (r *RexService) PublishToStream(msg *quorumpb.RumMsg, s network.Stream) error {

	remotePeer := s.Conn().RemotePeer()
	rumexchangelog.Debugf("PublishResponse msg to peer: %s", remotePeer)
	bufw := bufio.NewWriter(s)
	wc := protoio.NewDelimitedWriter(bufw)
	err := wc.WriteMsg(msg)
	if err != nil {
		rumexchangelog.Debugf("writemsg to network stream err: %s", err)
		metric.FailedCount.WithLabelValues(metric.ActionType.PublishToStream).Inc()
		return err
	} else {
		rumexchangelog.Debugf("writemsg to network stream succ: %s.", remotePeer)

		size := float64(metric.GetProtoSize(msg))
		metric.SuccessCount.WithLabelValues(metric.ActionType.PublishToStream).Inc()
		metric.OutBytes.WithLabelValues(metric.ActionType.PublishToStream).Set(size)
		metric.OutBytesTotal.WithLabelValues(metric.ActionType.PublishToStream).Add(size)
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

	bufw := bufio.NewWriter(s)
	wc := protoio.NewDelimitedWriter(bufw)
	err = wc.WriteMsg(msg)
	if err != nil {
		metric.FailedCount.WithLabelValues(metric.ActionType.PublishToPeerid).Inc()
		rumexchangelog.Debugf("writemsg to network stream err: %s", err)
		r.streampool.Delete(remotePeer)
		r.peerstore.AddIgnorePeer(toid)
		s.Close()

		return err
	} else {
		size := float64(metric.GetProtoSize(msg))
		metric.SuccessCount.WithLabelValues(metric.ActionType.PublishToPeerid).Inc()
		metric.OutBytes.WithLabelValues(metric.ActionType.PublishToPeerid).Set(size)
		metric.OutBytesTotal.WithLabelValues(metric.ActionType.PublishToPeerid).Add(size)

		rumexchangelog.Debugf("writemsg to network stream succ: %s.", to)
	}
	bufw.Flush()

	return nil
}

// Publish to 3 random connected peers
func (r *RexService) Publish(groupid string, msg *quorumpb.RumMsg) error {
	//TODO: select peers
	succ := 0
	peers := r.Host.Network().Peers()
	maxnum := 1

	//set timeout  and succ counter
	publishctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch := make(chan struct{})
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				ch <- struct{}{}
				return
			default:
				randompeerlist := r.peerstore.GetRandomPeer(groupid, maxnum, peers)
				for _, p := range randompeerlist {
					if err := r.PublishToPeerId(msg, peer.Encode(p)); err != nil {
						rumexchangelog.Debugf("writemsg to network stream err: %s", err)
					} else {
						succ++
						rumexchangelog.Debugf("writemsg to network stream succ: %s.", p)
					}
				}
			}

		}
	}(publishctx)

	<-ch
	return nil
}

// Publish to one random peer
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
	rumMsgSize := float64(metric.GetProtoSize(rummsg))
	switch rummsg.MsgType {
	case quorumpb.RumMsgType_RELAY_REQ, quorumpb.RumMsgType_RELAY_RESP:
		if rummsg.MsgType == quorumpb.RumMsgType_RELAY_REQ {
			metric.SuccessCount.WithLabelValues(metric.ActionType.RumRelayReq).Inc()
			metric.InBytes.WithLabelValues(metric.ActionType.RumRelayReq).Set(rumMsgSize)
			metric.InBytesTotal.WithLabelValues(metric.ActionType.RumRelayReq).Add(rumMsgSize)
		} else if rummsg.MsgType == quorumpb.RumMsgType_RELAY_RESP {
			metric.SuccessCount.WithLabelValues(metric.ActionType.RumRelayResp).Inc()
			metric.InBytes.WithLabelValues(metric.ActionType.RumRelayResp).Set(rumMsgSize)
			metric.InBytesTotal.WithLabelValues(metric.ActionType.RumRelayResp).Add(rumMsgSize)
		}

		for _, v := range r.msgtypehandlers {
			if v.Name == "rumrelay" {
				v.Handler(rummsg, s)
				break
			}
		}
	case quorumpb.RumMsgType_CHAIN_DATA:
		metric.SuccessCount.WithLabelValues(metric.ActionType.RumChainData).Inc()
		metric.InBytes.WithLabelValues(metric.ActionType.RumChainData).Set(rumMsgSize)
		metric.InBytesTotal.WithLabelValues(metric.ActionType.RumChainData).Add(rumMsgSize)

		for _, v := range r.msgtypehandlers {
			if v.Name == "rumchaindata" {
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
	rumexchangelog.Debugf("RumExchange stream handler %s start", remotePeer)
	defer rumexchangelog.Debugf("RumExchange stream handler %s exit", remotePeer)

	reader := msgio.NewVarintReaderSize(s, network.MessageSizeMax)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msgdata, err := reader.ReadMsg()
			if err != nil {
				if err != io.EOF {
					stat := s.Conn().Stat()
					rumexchangelog.Debugf("RumExchange stream handler from %s error: %s, stat: %v", s.Conn().RemotePeer(), err, stat)
					_ = s.Reset()
					return
				} else {
					rumexchangelog.Debugf("RumExchange stream handler EOF %s", remotePeer)
					r.streampool.Delete(remotePeer)
					_ = s.Close()
					return
				}
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
