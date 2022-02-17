package conn

import (
	"fmt"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/peer"
	iface "github.com/rumsystem/quorum/internal/pkg/chaindataciface"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/p2p"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"github.com/rumsystem/quorum/internal/pkg/pubsubconn"
	"google.golang.org/protobuf/proto"
)

var conn_log = logging.Logger("conn")

const (
	USER_CHANNEL_PREFIX     = "user_channel_"
	PRODUCER_CHANNEL_PREFIX = "prod_channel_"
	SYNC_CHANNEL_PREFIX     = "sync_channel_"
)

const (
	ERR_CAN_NOT_FIND_OWENR_PEER_ID = "ERR_CAN_NOT_FIND_OWENR_PEER_ID"
)

var conn *Conn

func GetConn() *Conn {
	return conn
}

type Conn struct {
	ConnMgrs map[string]*ConnMgr
}

type ConnMgr struct {
	GroupId               string
	UserChannelId         string
	ProducerChannelId     string
	SyncChannelId         string
	OwnerPubkey           string
	UserSignPubkey        string
	ProviderPeerIdPool    map[string]string
	ProducerPool          map[string]string
	StableProdPsConn      bool
	producerChannTimer    *time.Timer
	syncChannelTimersPool map[string]*time.Timer
	DataHandlerIface      iface.ChainDataHandlerIface
	PsConns               map[string]*pubsubconn.P2pPubSubConn
	Rex                   *p2p.RexService
}

type P2pNetworkType uint

const (
	PubSub P2pNetworkType = iota
	RumExchange
)

func (t P2pNetworkType) String() string {
	switch t {
	case PubSub:
		return "PubSub"
	case RumExchange:
		return "RumExchange"
	default:
		return fmt.Sprintf("%d", int(t))
	}
}

type PsConnChanel uint

const (
	UserChannel PsConnChanel = iota
	ProducerChannel
	SyncerChannel
)

func (t PsConnChanel) String() string {
	switch t {
	case UserChannel:
		return "UserChannel"
	case ProducerChannel:
		return "ProducerChannel"
	case SyncerChannel:
		return "SyncerChannel"
	default:
		return fmt.Sprintf("%d", int(t))
	}
}

const CLOSE_PRD_CHANN_TIMER time.Duration = 5  //5s
const CLOSE_SYNC_CHANN_TIMER time.Duration = 5 //5s

func InitConn() error {
	conn_log.Debug("Initconn called")
	conn = &Conn{}
	conn.ConnMgrs = make(map[string]*ConnMgr)
	return nil
}

func (conn *Conn) RegisterChainCtx(groupId, ownerPubkey, userSignPubkey string, cIface iface.ChainDataHandlerIface) error {
	conn_log.Debugf("RegisterChainCtx called, groupId <%S>", groupId)
	connMgr := &ConnMgr{}
	connMgr.InitGroupConnMgr(groupId, ownerPubkey, userSignPubkey, cIface)
	conn.ConnMgrs[groupId] = connMgr
	return nil
}

func (conn *Conn) UnregisterChainCtx(groupId string) error {
	conn_log.Debugf("UnregisterChainCtx called, groupId <%S>", groupId)

	connMgr, err := conn.GetConnMgr(groupId)

	if err != nil {
		return err
	}

	connMgr.LeaveAllChannels()
	//if in syncing, stop it

	//remove connMgr
	delete(conn.ConnMgrs, groupId)

	return nil
}

func (conn *Conn) GetConnMgr(groupId string) (*ConnMgr, error) {
	if connMgr, ok := conn.ConnMgrs[groupId]; ok {
		return connMgr, nil
	}
	return nil, fmt.Errorf("connMgr for group <%s> not exist", groupId)
}

func (connMgr *ConnMgr) InitGroupConnMgr(groupId string, ownerPubkey string, userSignPubkey string, cIface iface.ChainDataHandlerIface) error {
	conn_log.Debugf("InitGroupConnMgr called, groupId <%S>", groupId)
	connMgr.UserChannelId = USER_CHANNEL_PREFIX + groupId
	connMgr.ProducerChannelId = PRODUCER_CHANNEL_PREFIX + groupId
	connMgr.SyncChannelId = SYNC_CHANNEL_PREFIX + groupId + "_" + userSignPubkey
	connMgr.GroupId = groupId
	connMgr.OwnerPubkey = ownerPubkey
	connMgr.UserSignPubkey = userSignPubkey
	connMgr.ProviderPeerIdPool = make(map[string]string)
	connMgr.ProducerPool = make(map[string]string)
	connMgr.PsConns = make(map[string]*pubsubconn.P2pPubSubConn)

	connMgr.DataHandlerIface = cIface

	//Rex
	nodectx.GetNodeCtx().Node.RumExchange.ChainReg(connMgr.GroupId, cIface)

	//initial rex session
	connMgr.InitRexSession()

	//initial ps conn for user channel and sync channel
	connMgr.InitialPsConn()

	return nil
}

func (connMgr *ConnMgr) UpdateProviderPeerIdPool(peerPubkey, peerId string) error {
	conn_log.Debug("UpdateProviderPeerIdPool called, groupId <%S>", connMgr.GroupId)
	connMgr.ProviderPeerIdPool[peerPubkey] = peerId
	return connMgr.InitRexSession()
}

func (connMgr *ConnMgr) UpdProducers(pubkeys []string) error {
	conn_log.Debug("AddProducer, groupId <%S>", connMgr.GroupId)
	connMgr.ProducerPool = make(map[string]string)

	for _, pubkey := range pubkeys {
		connMgr.ProducerPool[pubkey] = pubkey
	}

	if _, ok := connMgr.ProducerPool[connMgr.UserSignPubkey]; ok {
		conn_log.Debug("I am producer, create producer psconn, groupId <%S>", connMgr.GroupId)
		connMgr.StableProdPsConn = true
		connMgr.getProducerPsConn()
	} else {
		conn_log.Debug("I am NOT producer, create producer psconn only when needed, groupId <%S>", connMgr.GroupId)
		connMgr.StableProdPsConn = false
	}

	return nil
}

func (connMgr *ConnMgr) InitRexSession() error {
	conn_log.Debug("InitSession called, groupId <%S>", connMgr.GroupId)
	if peerId, ok := connMgr.ProviderPeerIdPool[connMgr.OwnerPubkey]; ok {
		err := nodectx.GetNodeCtx().Node.RumExchange.InitSession(peerId, connMgr.ProducerChannelId)
		if err != nil {
			return err
		}
		err = nodectx.GetNodeCtx().Node.RumExchange.InitSession(peerId, connMgr.SyncChannelId)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf(ERR_CAN_NOT_FIND_OWENR_PEER_ID)
	}
	return nil
}

func (connMgr *ConnMgr) LeaveAllChannels() error {
	conn_log.Debug("LeaveChannel called, groupId <%S>", connMgr.GroupId)
	for channelId, _ := range connMgr.PsConns {
		nodectx.GetNodeCtx().Node.PubSubConnMgr.LeaveChannel(channelId)
		delete(connMgr.PsConns, channelId)
	}
	return nil
}

func (connMgr *ConnMgr) getProducerPsConn() *pubsubconn.P2pPubSubConn {
	conn_log.Debugf("<%s> GetProducerTrxMgr called", connMgr.GroupId)

	if psconn, ok := connMgr.PsConns[connMgr.ProducerChannelId]; ok {
		if !connMgr.StableProdPsConn { //is user, no need to keep producer psconn
			conn_log.Debugf("<%s> reset connection timer for producer psconn <%s>", connMgr.GroupId, connMgr.ProducerChannelId)
			connMgr.producerChannTimer.Stop()
			connMgr.producerChannTimer.Reset(CLOSE_PRD_CHANN_TIMER * time.Second)
		}
		return psconn
	} else {
		producerPsconn := nodectx.GetNodeCtx().Node.PubSubConnMgr.GetPubSubConnByChannelId(connMgr.ProducerChannelId, connMgr.DataHandlerIface)
		connMgr.PsConns[connMgr.ProducerChannelId] = producerPsconn
		if !connMgr.StableProdPsConn {
			conn_log.Debugf("<%s> create close_conn timer for producer channel <%s>", connMgr.GroupId, connMgr.ProducerChannelId)
			connMgr.producerChannTimer = time.AfterFunc(CLOSE_PRD_CHANN_TIMER*time.Second, func() {
				conn_log.Debugf("<%s> time up, close producer channel <%s>", connMgr.GroupId, connMgr.ProducerChannelId)
				nodectx.GetNodeCtx().Node.PubSubConnMgr.LeaveChannel(connMgr.ProducerChannelId)
				delete(connMgr.PsConns, connMgr.ProducerChannelId)
			})
		}
		return producerPsconn
	}
}

func (connMgr *ConnMgr) getSyncConn(channelId string) (*pubsubconn.P2pPubSubConn, error) {
	conn_log.Debugf("<%s> getSyncConn called", connMgr.GroupId)

	if psconn, ok := connMgr.PsConns[channelId]; ok {
		conn_log.Debugf("<%s> reset connection timer for syncer psconn <%s>", connMgr.GroupId, channelId)
		if timer, ok := connMgr.syncChannelTimersPool[channelId]; ok {
			timer.Stop()
			timer.Reset(CLOSE_SYNC_CHANN_TIMER * time.Second)
		} else {
			return nil, fmt.Errorf("Can not find timer for syncer channel")
		}
		return psconn, nil
	} else {
		syncerPsconn := nodectx.GetNodeCtx().Node.PubSubConnMgr.GetPubSubConnByChannelId(channelId, connMgr.DataHandlerIface)
		connMgr.PsConns[channelId] = syncerPsconn
		conn_log.Debugf("<%s> create close_conn timer for syncer channel <%s>", connMgr.GroupId, channelId)
		syncTimer := time.AfterFunc(CLOSE_PRD_CHANN_TIMER*time.Second, func() {
			conn_log.Debugf("<%s> time up, close syncer channel <%s>", connMgr.GroupId, channelId)
			nodectx.GetNodeCtx().Node.PubSubConnMgr.LeaveChannel(connMgr.ProducerChannelId)
			delete(connMgr.PsConns, channelId)
			delete(connMgr.syncChannelTimersPool, channelId)
		})
		connMgr.syncChannelTimersPool[channelId] = syncTimer
		return syncerPsconn, nil
	}
}

func (connMgr *ConnMgr) getUserConn() *pubsubconn.P2pPubSubConn {
	return connMgr.PsConns[connMgr.UserChannelId]
}

func (connMgr *ConnMgr) SendBlockPsconn(blk *quorumpb.Block, psChannel PsConnChanel, chanelId ...string) error {
	var pkg *quorumpb.Package
	pkg = &quorumpb.Package{}

	pbBytes, err := proto.Marshal(blk)
	if err != nil {
		return err
	}

	pkg.Type = quorumpb.PackageType_BLOCK
	pkg.Data = pbBytes

	pkgBytes, err := proto.Marshal(pkg)
	if err != nil {
		return err
	}

	if psChannel == ProducerChannel {
		psconn := connMgr.getProducerPsConn()
		return psconn.Publish(pkgBytes)
	} else if psChannel == UserChannel {
		psconn := connMgr.getUserConn()
		return psconn.Publish(pkgBytes)
	} else if psChannel == SyncerChannel {
		psconn, err := connMgr.getSyncConn(chanelId[0])
		if err != nil {
			return err
		}

		return psconn.Publish(pkgBytes)
	}

	return fmt.Errorf("Can not find psChannel")
}

func (connMgr *ConnMgr) SendBlockRex(blk *quorumpb.Block) error {
	var pkg *quorumpb.Package
	pkg = &quorumpb.Package{}

	pbBytes, err := proto.Marshal(blk)
	if err != nil {
		return err
	}

	pkg.Type = quorumpb.PackageType_BLOCK
	pkg.Data = pbBytes

	rummsg := &quorumpb.RumMsg{MsgType: quorumpb.RumMsgType_CHAIN_DATA, DataPackage: pkg}
	return connMgr.Rex.Publish(rummsg)

}

func (connMgr *ConnMgr) SendTrxPubsub(trx *quorumpb.Trx, psChannel PsConnChanel) error {
	var pkg *quorumpb.Package
	pkg = &quorumpb.Package{}

	pbBytes, err := proto.Marshal(trx)
	if err != nil {
		return err
	}

	pkg.Type = quorumpb.PackageType_TRX
	pkg.Data = pbBytes

	pkgBytes, err := proto.Marshal(pkg)
	if err != nil {
		return err
	}

	if psChannel == ProducerChannel {
		psconn := connMgr.getProducerPsConn()
		return psconn.Publish(pkgBytes)
	}

	/*
		if psChannel == SyncerChannel {
			psconn := connMgr.getSyncConn()
			return psconn.Publish(pkgBytes)
		}
	*/

	if psconn, ok := connMgr.PsConns[PsConnChanel.String(psChannel)]; ok {
		return psconn.Publish(pkgBytes)
	}

	return fmt.Errorf("Can not find psChannel")
}

func (connMgr *ConnMgr) SendTrxRex(trx *quorumpb.Trx, to peer.ID) error {
	var pkg *quorumpb.Package
	pkg = &quorumpb.Package{}

	pbBytes, err := proto.Marshal(trx)
	if err != nil {
		return err
	}

	pkg.Type = quorumpb.PackageType_TRX
	pkg.Data = pbBytes
	rummsg := &quorumpb.RumMsg{MsgType: quorumpb.RumMsgType_CHAIN_DATA, DataPackage: pkg}
	return nodectx.GetNodeCtx().Node.RumExchange.PublishTo(rummsg, to)
}

func (connMgr *ConnMgr) InitialPsConn() {
	userPsconn := nodectx.GetNodeCtx().Node.PubSubConnMgr.GetPubSubConnByChannelId(connMgr.UserChannelId, connMgr.DataHandlerIface)
	connMgr.PsConns[connMgr.UserChannelId] = userPsconn
	syncerPsconn := nodectx.GetNodeCtx().Node.PubSubConnMgr.GetPubSubConnByChannelId(connMgr.SyncChannelId, connMgr.DataHandlerIface)
	connMgr.PsConns[connMgr.SyncChannelId] = syncerPsconn
}
