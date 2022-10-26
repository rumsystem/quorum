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
	pubsubconn "github.com/rumsystem/quorum/internal/pkg/conn/pubsubconn"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/metric"
	"github.com/rumsystem/quorum/pkg/constants"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var rumexchangelog = logging.Logger("rumexchange")
var peerstoreTTL time.Duration = time.Duration(20 * time.Minute)

const IDVer = "2.0.0"

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
	Host          host.Host
	pubSubConnMgr *pubsubconn.PubSubConnMgr
	//peerStatus         *PeerStatus
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

//type streamPoolItem struct {
//	s      network.Stream
//	cancel context.CancelFunc
//}

func NewRexService(h host.Host, psconnmgr *pubsubconn.PubSubConnMgr, Networkname string, ProtocolPrefix string, notification chan RexNotification) *RexService {
	customprotocol := fmt.Sprintf("%s/%s/rex/%s", ProtocolPrefix, Networkname, IDVer)
	chainmgr := make(map[string]chaindef.ChainDataSyncIface)
	//rumpeerstore := &RumGroupPeerStore{}
	rumpeerstore := NewRumGroupPeerStore()
	rexs := &RexService{Host: h, pubSubConnMgr: psconnmgr, peerstore: rumpeerstore, ProtocolId: protocol.ID(customprotocol), notificationch: notification, chainmgr: chainmgr}
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

func (r *RexService) NewStream(peerid peer.ID) (network.Stream, error) {
	//only request trx need to create new stream, so a handler gorutine will be create to waiting the resp.
	//TODO:  the ctx will timeout after x sec.

	//new stream
	//ctx, _ := context.WithCancel(context.Background())
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	//TODO return cancel
	//defer cancel()

	// could be a transient stream(relay)
	s, err := r.Host.NewStream(ctx, peerid, r.ProtocolId)
	//newpoolitem := &streamPoolItem{s: s, cancel: cancel}
	if err != nil {
		return nil, err
	}
	//r.streampool.Store(peerid, newpoolitem)

	go r.HandlerProcessStream(ctx, s)

	return s, nil
}

func (r *RexService) ChainReg(groupid string, cdhIface chaindef.ChainDataSyncIface) {
	_, ok := r.chainmgr[groupid]
	if ok == false {
		r.chainmgr[groupid] = cdhIface
		rumexchangelog.Debugf("chain reg with rumexchange: %s", groupid)
	}
}

func (r *RexService) PublishToStream(msg *quorumpb.RumMsg, s network.Stream) error {
	//TODO:  add a timeout ctx to close the steam after timeout
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

	s, err := r.NewStream(toid)
	if err != nil {
		rumexchangelog.Debugf("create network stream to %s err: %s", to, err)
		return err
	}
	//s := poolitem.s
	//remotePeer := s.Conn().RemotePeer()

	bufw := bufio.NewWriter(s)
	wc := protoio.NewDelimitedWriter(bufw)
	err = wc.WriteMsg(msg)
	if err != nil {
		metric.FailedCount.WithLabelValues(metric.ActionType.PublishToPeerid).Inc()
		rumexchangelog.Debugf("writemsg to network stream err: %s", err)
		r.peerstore.Scorers().BadResponsesScorer().Increment(toid)
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

// Publish to 1 random connected peers
func (r *RexService) Publish(groupid string, msg *quorumpb.RumMsg) error {
	//TODO: save good peers?
	ctx := context.Background()
	connectedpeers := r.Host.Network().Peers()
	UserChannelId := constants.USER_CHANNEL_PREFIX + groupid
	channelpeers, err := r.pubSubConnMgr.GetPeersByChannelId(UserChannelId)
	if err == nil {
		if len(channelpeers) > 0 {
			connectedpeers = channelpeers
		}
	}
	peers := r.peerstore.filterPeers(ctx, connectedpeers, 0.7)

	//TODO: CLOSE the stream before return? (defer?)
	//publishctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	//defer cancel()

	for _, p := range peers {
		if err := r.PublishToPeerId(msg, peer.Encode(p)); err == nil {
			r.peerstore.Scorers().BlockProviderScorer().Touch(p)
			rumexchangelog.Debugf("writemsg to network stream succ: %s.", p)
			return nil
		} else {
			r.peerstore.Scorers().BadResponsesScorer().Increment(p)
			rumexchangelog.Debugf("writemsg to network stream err: %s", err)
		}
	}

	return rumerrors.ErrNoPeersAvailable

	return nil
}

func (r *RexService) HandleRumExchangeMsg(rummsg *quorumpb.RumMsg, s network.Stream) {
	rumMsgSize := float64(metric.GetProtoSize(rummsg))
	switch rummsg.MsgType {
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
	r.HandlerProcessStream(ctx, s)
}

func (r *RexService) HandlerProcessStream(ctx context.Context, s network.Stream) {
	remotePeer := s.Conn().RemotePeer()
	rumexchangelog.Debugf("RumExchange stream handler %s start", remotePeer)
	defer func() {
		rumexchangelog.Debugf("RumExchange stream handler %s exit", remotePeer)
		_ = s.Close()
	}()

	reader := msgio.NewVarintReaderSize(s, network.MessageSizeMax)
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
				//r.streampool.Delete(remotePeer)
				return
			}
		}
		var rummsg quorumpb.RumMsg
		if err = proto.Unmarshal(msgdata, &rummsg); err == nil {
			r.HandleRumExchangeMsg(&rummsg, s)
		}
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
