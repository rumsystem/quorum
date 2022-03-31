package conn

import (
	"errors"
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
	SyncChannelTimersPool map[string]*time.Timer
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

const CLOSE_PRD_CHANN_TIMER time.Duration = 20  //5s
const CLOSE_SYNC_CHANN_TIMER time.Duration = 20 //5s

func InitConn() error {
	conn_log.Debug("Initconn called")
	conn = &Conn{}
	conn.ConnMgrs = make(map[string]*ConnMgr)
	return nil
}

func (conn *Conn) RegisterChainCtx(groupId, ownerPubkey, userSignPubkey string, cIface iface.ChainDataHandlerIface) error {
	conn_log.Debugf("RegisterChainCtx called, groupId <%s>", groupId)
	connMgr := &ConnMgr{}
	connMgr.InitGroupConnMgr(groupId, ownerPubkey, userSignPubkey, cIface)
	conn.ConnMgrs[groupId] = connMgr
	return nil
}

func (conn *Conn) UnregisterChainCtx(groupId string) error {
	conn_log.Debugf("UnregisterChainCtx called, groupId <%s>", groupId)

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
	conn_log.Debugf("InitGroupConnMgr called, groupId <%s>", groupId)
	connMgr.UserChannelId = USER_CHANNEL_PREFIX + groupId
	connMgr.ProducerChannelId = PRODUCER_CHANNEL_PREFIX + groupId
	connMgr.SyncChannelId = SYNC_CHANNEL_PREFIX + groupId + "_" + userSignPubkey
	connMgr.GroupId = groupId
	connMgr.OwnerPubkey = ownerPubkey
	connMgr.UserSignPubkey = userSignPubkey
	connMgr.ProviderPeerIdPool = make(map[string]string)
	connMgr.ProducerPool = make(map[string]string)
	connMgr.PsConns = make(map[string]*pubsubconn.P2pPubSubConn)
	connMgr.SyncChannelTimersPool = make(map[string]*time.Timer)

	connMgr.DataHandlerIface = cIface

	//Rex
	//nodectx.GetNodeCtx().Node.RumExchange.ChainReg(connMgr.GroupId, cIface)

	if nodectx.GetNodeCtx().Node.RumExchange != nil {
		nodectx.GetNodeCtx().Node.RumExchange.ChainReg(connMgr.GroupId, cIface)
	}

	//initial rex session
	//connMgr.InitRexSession()

	//initial ps conn for user channel and sync channel
	connMgr.InitialPsConn()

	return nil
}

func (connMgr *ConnMgr) UpdateProviderPeerIdPool(peerPubkey, peerId string) error {
	conn_log.Debugf("UpdateProviderPeerIdPool called, groupId <%s>", connMgr.GroupId)
	connMgr.ProviderPeerIdPool[peerPubkey] = peerId
	return connMgr.InitRexSession()
}

func (connMgr *ConnMgr) UpdProducers(pubkeys []string) error {
	conn_log.Debugf("AddProducer, groupId <%s>", connMgr.GroupId)
	connMgr.ProducerPool = make(map[string]string)

	for _, pubkey := range pubkeys {
		connMgr.ProducerPool[pubkey] = pubkey
	}

	if _, ok := connMgr.ProducerPool[connMgr.UserSignPubkey]; ok {
		conn_log.Debugf("I am producer, create producer psconn, groupId <%s>", connMgr.GroupId)
		connMgr.StableProdPsConn = true
		connMgr.getProducerPsConn()
	} else {
		conn_log.Debugf("I am NOT producer, create producer psconn only when needed, groupId <%s>", connMgr.GroupId)
		connMgr.StableProdPsConn = false
	}

	return nil
}

func (connMgr *ConnMgr) InitRexSession() error {
	conn_log.Debugf("InitSession called, groupId <%s>", connMgr.GroupId)
	if peerId, ok := connMgr.ProviderPeerIdPool[connMgr.OwnerPubkey]; ok {
		if nodectx.GetNodeCtx().Node.RumSession == nil {
			return nil
		}
		err := nodectx.GetNodeCtx().Node.RumSession.InitSession(peerId, connMgr.ProducerChannelId)
		if err != nil {
			return err
		}
		err = nodectx.GetNodeCtx().Node.RumSession.InitSession(peerId, connMgr.SyncChannelId)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf(ERR_CAN_NOT_FIND_OWENR_PEER_ID)
	}
	return nil
}

func (connMgr *ConnMgr) LeaveAllChannels() error {
	conn_log.Debugf("LeaveChannel called, groupId <%s>", connMgr.GroupId)
	for channelId, _ := range connMgr.PsConns {
		nodectx.GetNodeCtx().Node.PubSubConnMgr.LeaveChannel(channelId)
		delete(connMgr.PsConns, channelId)
	}
	return nil
}

func (connMgr *ConnMgr) getProducerPsConn() *pubsubconn.P2pPubSubConn {
	//conn_log.Debugf("<%s> getProducerPsConn called", connMgr.GroupId)
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
	//conn_log.Debugf("<%s> getSyncConn called", connMgr.GroupId)
	if psconn, ok := connMgr.PsConns[channelId]; ok {
		conn_log.Debugf("<%s> reset connection timer for syncer psconn <%s>", connMgr.GroupId, channelId)
		if timer, ok := connMgr.SyncChannelTimersPool[channelId]; ok {
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
			nodectx.GetNodeCtx().Node.PubSubConnMgr.LeaveChannel(channelId)
			delete(connMgr.PsConns, channelId)
			delete(connMgr.SyncChannelTimersPool, channelId)
		})
		connMgr.SyncChannelTimersPool[channelId] = syncTimer
		return syncerPsconn, nil
	}
}

func (connMgr *ConnMgr) getUserConn() *pubsubconn.P2pPubSubConn {
	//conn_log.Debugf("<%s> getUserConn called", connMgr.GroupId)
	return connMgr.PsConns[connMgr.UserChannelId]
}

func (connMgr *ConnMgr) SendBlockPsconn(blk *quorumpb.Block, psChannel PsConnChanel, chanelId ...string) error {
	conn_log.Debugf("<%s> SendBlockPsconn called", connMgr.GroupId)
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
		conn_log.Debugf("<%s> Send block via Producer_Channel", connMgr.GroupId)
		psconn := connMgr.getProducerPsConn()
		return psconn.Publish(pkgBytes)
	} else if psChannel == UserChannel {
		conn_log.Debugf("<%s> Send block via User_Channel", connMgr.GroupId)
		psconn := connMgr.getUserConn()
		return psconn.Publish(pkgBytes)
	} else if psChannel == SyncerChannel {
		conn_log.Debugf("<%s> Send block via Syncer_Channel <%s>", connMgr.GroupId, chanelId[0])
		psconn, err := connMgr.getSyncConn(chanelId[0])
		if err != nil {
			return err
		}
		return psconn.Publish(pkgBytes)
	}

	return fmt.Errorf("Can not find psChannel")
}

func (connMgr *ConnMgr) SendSnapshotPsconn(snapshot *quorumpb.Snapshot, psChannel PsConnChanel, chanelId ...string) error {
	conn_log.Debugf("<%s> SendSnapshotPsconn called", connMgr.GroupId)

	var pkg *quorumpb.Package
	pkg = &quorumpb.Package{}

	pbBytes, err := proto.Marshal(snapshot)
	if err != nil {
		return err
	}

	pkg.Type = quorumpb.PackageType_SNAPSHOT
	pkg.Data = pbBytes

	pkgBytes, err := proto.Marshal(pkg)
	if err != nil {
		return err
	}

	if psChannel == ProducerChannel {
		conn_log.Debugf("<%s> Send snapshot via Producer_Channel", connMgr.GroupId)
		psconn := connMgr.getProducerPsConn()
		return psconn.Publish(pkgBytes)
	} else if psChannel == UserChannel {
		conn_log.Debugf("<%s> Send snapshot via User_Channel", connMgr.GroupId)
		psconn := connMgr.getUserConn()
		return psconn.Publish(pkgBytes)
	} else if psChannel == SyncerChannel {
		conn_log.Debugf("<%s> Send snapshot via Syncer_Channel <%s>", connMgr.GroupId, chanelId[0])
		psconn, err := connMgr.getSyncConn(chanelId[0])
		if err != nil {
			return err
		}
		return psconn.Publish(pkgBytes)
	}

	return fmt.Errorf("Can not find psChannel")
}

func (connMgr *ConnMgr) SendBlockRex(blk *quorumpb.Block) error {
	conn_log.Debugf("<%s> SendBlockRex called", connMgr.GroupId)
	var pkg *quorumpb.Package
	pkg = &quorumpb.Package{}

	pbBytes, err := proto.Marshal(blk)
	if err != nil {
		return err
	}

	pkg.Type = quorumpb.PackageType_BLOCK
	pkg.Data = pbBytes

	rummsg := &quorumpb.RumMsg{MsgType: quorumpb.RumMsgType_CHAIN_DATA, DataPackage: pkg}
	return connMgr.Rex.Publish(blk.GroupId, rummsg)

}

func (connMgr *ConnMgr) SendTrxPubsub(trx *quorumpb.Trx, psChannel PsConnChanel, channelId ...string) error {
	conn_log.Debugf("<%s> SendTrxPubsub called", connMgr.GroupId)
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
		conn_log.Debugf("<%s> Send trx via Producer_Channel", connMgr.GroupId)
		psconn := connMgr.getProducerPsConn()
		return psconn.Publish(pkgBytes)
	} else if psChannel == UserChannel {
		conn_log.Debugf("<%s> Send trx via User_Channel", connMgr.GroupId)
		psconn := connMgr.getUserConn()
		return psconn.Publish(pkgBytes)
	} else if psChannel == SyncerChannel {
		conn_log.Debugf("<%s> Send trx via Syncer_Channel <%s>", connMgr.GroupId, channelId[0])
		psconn, err := connMgr.getSyncConn(channelId[0])
		if err != nil {
			return err
		}
		return psconn.Publish(pkgBytes)
	}

	return fmt.Errorf("Can not find psChannel")
}

func (connMgr *ConnMgr) SendTrxRex(trx *quorumpb.Trx, to peer.ID) error {
	conn_log.Debugf("<%s> SendTrxRex called", connMgr.GroupId)
	if nodectx.GetNodeCtx().Node.RumExchange == nil {
		return errors.New("RumExchange is nil, please set enablerumexchange as true")
	}

	var pkg *quorumpb.Package
	pkg = &quorumpb.Package{}

	pbBytes, err := proto.Marshal(trx)
	if err != nil {
		return err
	}

	pkg.Type = quorumpb.PackageType_TRX
	pkg.Data = pbBytes
	rummsg := &quorumpb.RumMsg{MsgType: quorumpb.RumMsgType_CHAIN_DATA, DataPackage: pkg}
	if to == "" {
		return nodectx.GetNodeCtx().Node.RumExchange.Publish(trx.GroupId, rummsg)
	} else {
		return nodectx.GetNodeCtx().Node.RumExchange.PublishTo(rummsg, to)
	}
}

func (connMgr *ConnMgr) InitialPsConn() {
	conn_log.Debugf("<%s> InitialPsConn called", connMgr.GroupId)
	userPsconn := nodectx.GetNodeCtx().Node.PubSubConnMgr.GetPubSubConnByChannelId(connMgr.UserChannelId, connMgr.DataHandlerIface)
	connMgr.PsConns[connMgr.UserChannelId] = userPsconn
	syncerPsconn := nodectx.GetNodeCtx().Node.PubSubConnMgr.GetPubSubConnByChannelId(connMgr.SyncChannelId, connMgr.DataHandlerIface)
	connMgr.PsConns[connMgr.SyncChannelId] = syncerPsconn
}
