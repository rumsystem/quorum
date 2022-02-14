package nodectx

import (
	"fmt"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/peer"
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

type ChainDataHandlerIface interface {
	HandleTrxPsConn(trx *quorumpb.Trx) error
	HandleBlockPsConn(block *quorumpb.Block) error
	HandleTrxRex(trx *quorumpb.Trx, from peer.ID) error
	HandleBlockRex(block *quorumpb.Block, from peer.ID) error
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
	DataHandlerIface      ChainDataHandlerIface
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

func (conn *Conn) RegisterChainCtx(groupId, ownerPubkey, userSignPubkey string, cIface ChainDataHandlerIface) error {
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

func (connMgr *ConnMgr) InitGroupConnMgr(groupId string, ownerPubkey string, userSignPubkey string, cIface ChainDataHandlerIface) error {
	conn_log.Debugf("InitGroupConnMgr called, groupId <%S>", groupId)
	connMgr.UserChannelId = USER_CHANNEL_PREFIX + groupId
	connMgr.ProducerChannelId = PRODUCER_CHANNEL_PREFIX + groupId
	connMgr.SyncChannelId = SYNC_CHANNEL_PREFIX + groupId + "_" + userSignPubkey
	connMgr.GroupId = groupId
	connMgr.OwnerPubkey = ownerPubkey
	connMgr.UserSignPubkey = userSignPubkey
	connMgr.ProviderPeerIdPool = make(map[string]string)
	connMgr.ProducerPool = make(map[string]string)

	connMgr.DataHandlerIface = cIface

	//Rex
	GetNodeCtx().Node.RumExchange.ChainReg(connMgr.GroupId, cIface)

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

func (connMgr *ConnMgr) AddProducer(peerPubkey string) error {
	conn_log.Debug("AddProducer, groupId <%S>", connMgr.GroupId)
	connMgr.ProducerPool[peerPubkey] = peerPubkey
	if peerPubkey == connMgr.UserSignPubkey {
		connMgr.StableProdPsConn = true
	} else {
		connMgr.StableProdPsConn = false
	}

	return nil
}

func (connMgr *ConnMgr) RmProducer(peerPubkey string) error {
	conn_log.Debug("RmProducer, groupId <%S>", connMgr.GroupId)
	if _, ok := connMgr.ProducerPool[peerPubkey]; ok {
		delete(connMgr.ProducerPool, peerPubkey)
		if peerPubkey == connMgr.UserSignPubkey {
			connMgr.StableProdPsConn = false
		}
	} else {
		return fmt.Errorf("key not exist")
	}
	return nil
}

func (connMgr *ConnMgr) InitRexSession() error {
	conn_log.Debug("InitSession called, groupId <%S>", connMgr.GroupId)
	if peerId, ok := connMgr.ProviderPeerIdPool[connMgr.OwnerPubkey]; ok {
		err := GetNodeCtx().Node.RumExchange.InitSession(peerId, connMgr.ProducerChannelId)
		if err != nil {
			return err
		}
		err = GetNodeCtx().Node.RumExchange.InitSession(peerId, connMgr.SyncChannelId)
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
		GetNodeCtx().Node.PubSubConnMgr.LeaveChannel(channelId)
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
		producerPsconn := GetNodeCtx().Node.PubSubConnMgr.GetPubSubConnByChannelId(connMgr.ProducerChannelId, connMgr.DataHandlerIface)
		connMgr.PsConns[connMgr.ProducerChannelId] = producerPsconn
		if !connMgr.StableProdPsConn {
			conn_log.Debugf("<%s> create close_conn timer for producer channel <%s>", connMgr.GroupId, connMgr.ProducerChannelId)
			connMgr.producerChannTimer = time.AfterFunc(CLOSE_PRD_CHANN_TIMER*time.Second, func() {
				conn_log.Debugf("<%s> time up, close producer channel <%s>", connMgr.GroupId, connMgr.ProducerChannelId)
				GetNodeCtx().Node.PubSubConnMgr.LeaveChannel(connMgr.ProducerChannelId)
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
		syncerPsconn := GetNodeCtx().Node.PubSubConnMgr.GetPubSubConnByChannelId(channelId, connMgr.DataHandlerIface)
		connMgr.PsConns[channelId] = syncerPsconn
		conn_log.Debugf("<%s> create close_conn timer for syncer channel <%s>", connMgr.GroupId, channelId)
		syncTimer := time.AfterFunc(CLOSE_PRD_CHANN_TIMER*time.Second, func() {
			conn_log.Debugf("<%s> time up, close syncer channel <%s>", connMgr.GroupId, channelId)
			GetNodeCtx().Node.PubSubConnMgr.LeaveChannel(connMgr.ProducerChannelId)
			delete(connMgr.PsConns, channelId)
			delete(connMgr.syncChannelTimersPool, channelId)
		})
		connMgr.syncChannelTimersPool[channelId] = syncTimer
		return syncerPsconn, nil
	}
}

func (connMgr *ConnMgr) SendBlock(blk *quorumpb.Block, networktype P2pNetworkType, psChannel ...PsConnChanel) error {

	var pkg *quorumpb.Package
	pkg = &quorumpb.Package{}

	pbBytes, err := proto.Marshal(blk)
	if err != nil {
		return err
	}

	pkg.Type = quorumpb.PackageType_BLOCK
	pkg.Data = pbBytes

	//TODO: rex or pubsub
	if networktype == RumExchange {
		rummsg := &quorumpb.RumMsg{MsgType: quorumpb.RumMsgType_CHAIN_DATA, DataPackage: pkg}
		return connMgr.Rex.Publish(rummsg)
	}

	pkgBytes, err := proto.Marshal(pkg)
	if err != nil {
		return err
	}

	if psChannel[0] == ProducerChannel {
		psconn := connMgr.getProducerPsConn()
		return psconn.Publish(pkgBytes)
	}

	if psconn, ok := connMgr.PsConns[PsConnChanel.String(psChannel[0])]; ok {
		return psconn.Publish(pkgBytes)
	}

	return fmt.Errorf("Can not find psChannel")
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
	return GetNodeCtx().Node.RumExchange.PublishTo(rummsg, to)
}

func (connMgr *ConnMgr) InitialPsConn() {
	userPsconn := GetNodeCtx().Node.PubSubConnMgr.GetPubSubConnByChannelId(connMgr.UserChannelId, connMgr.DataHandlerIface)
	connMgr.PsConns[connMgr.UserChannelId] = userPsconn
	syncerPsconn := GetNodeCtx().Node.PubSubConnMgr.GetPubSubConnByChannelId(connMgr.UserChannelId, connMgr.DataHandlerIface)
	connMgr.PsConns[connMgr.SyncChannelId] = syncerPsconn
}

func (connMgr *ConnMgr) SyncForward(blockId string, nodename string) error {
	conn_log.Debug("SyncForward called, groupId <%S>", connMgr.GroupId)
	/*
		topBlock, err := nodectx.GetDbMgr().GetBlock(blockId, false, nodename)
		if err != nil {
			conn_log.Warningf("Get top block error, blockId <%s> at <%s>, <%s>", higestBId, grp.ChainCtx.nodename, err.Error())
			return err
		}

		if chain.Syncer != nil {
			return chain.Syncer.SyncForward(block)
		}
		return nil
	*/
	return nil
}

func (connMgr *ConnMgr) SyncBackward(blockId string, nodename string) error {
	conn_log.Debug("SyncBackward called, groupId <%S>", connMgr.GroupId)
	/*
		topBlock, err := nodectx.GetDbMgr().GetBlock(blockId, false, nodename)
		if err != nil {
			conn_log.Warningf("Get top block error, blockId <%s> at <%s>, <%s>", higestBId, grp.ChainCtx.nodename, err.Error())
			return err
		}

		if chain.Syncer != nil {
			return chain.Syncer.SyncBackward(block)
		}

		return nil
	*/

	return nil
}

func (connMgr *ConnMgr) StartInitialSync(block *quorumpb.Block) error {
	conn_log.Debugf("<%s> StartInitialSync called", connMgr.GroupId)

	/*
		if chain.Syncer != nil {
			return chain.Syncer.SyncForward(block)
		}
		return nil
	*/

	return nil
}

func (connMgr *ConnMgr) StopSync() error {
	conn_log.Debugf("<%s> StopSync called", connMgr.GroupId)
	/*
		if connMgr.Syncer != nil {
			return connMgr.Syncer.StopSync()
		}
		return nil
	*/

	return nil
}

func (connMgr *ConnMgr) AddBlockSynced(resp *quorumpb.ReqBlockResp, block *quorumpb.Block) error {
	/*
		if connMgr.Syncer != nil {
			return connMgr.Syncer.AddBlockSynced(resp, block)
		}
		return nil
	*/

	return nil
}

func (connMgr *ConnMgr) IsSyncerReady() bool {
	conn_log.Debugf("<%s> IsSyncerReady called", connMgr.GroupId)
	/*
		if chain.Syncer.Status == SYNCING_BACKWARD ||
			chain.Syncer.Status == SYNCING_FORWARD ||
			chain.Syncer.Status == SYNC_FAILED {
			chain_log.Debugf("<%s> syncer is busy, status: <%d>", chain.groupId, chain.Syncer.Status)
			return true
		}
		chain_log.Debugf("<%s> syncer is IDLE", chain.groupId)
		return false
	*/

	return false
}

//channelId := SYNC_CHANNEL_PREFIX + producer.grpItem.GroupId + "_" + reqBlockItem.UserId

/*
if producer.cIface.IsSyncerReady() {
	return
}
*/
