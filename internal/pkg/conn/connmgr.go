package conn

import (
	"context"
	"errors"
	"fmt"
	"time"

	logging "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/network"
	chaindef "github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/internal/pkg/conn/pubsubconn"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/pkg/constants"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var conn_log = logging.Logger("conn")

var conn *Conn

func GetConn() *Conn {
	return conn
}

type Conn struct {
	ConnMgrs map[string]*ConnMgr // key: groupId or groupId
}

type ConnMgr struct {
	GroupId            string
	UserChannelId      string
	ProducerChannelId  string
	OwnerPubkey        string
	UserSignPubkey     string
	ProviderPeerIdPool map[string]string // key: group owner PubKey; value: group owner peerId
	ProducerPool       map[string]string // key: group producer Pubkey; value: group producer Pubkey
	StableProdPsConn   bool
	producerChannTimer *time.Timer
	DataHandlerIface   chaindef.ChainDataSyncIface
	//TODO: sync.map
	ps      *pubsub.PubSub
	PsConns map[string]*pubsubconn.P2pPubSubConn // key: channelId
	//Rex     *p2p.RexService
}

type PsConnChanel uint

const (
	UserChannel PsConnChanel = iota
	ProducerChannel
)

func (t PsConnChanel) String() string {
	switch t {
	case UserChannel:
		return "UserChannel"
	case ProducerChannel:
		return "ProducerChannel"
	default:
		return fmt.Sprintf("%d", int(t))
	}
}

const (
	CLOSE_PRD_CHANN_TIMER time.Duration = 20 * time.Second
)

func InitConn() error {
	conn_log.Debug("Initconn called")
	conn = &Conn{}
	conn.ConnMgrs = make(map[string]*ConnMgr)
	return nil
}

func (conn *Conn) RegisterChainCtx(groupId, ownerPubkey, userSignPubkey string, cIface chaindef.ChainDataSyncIface) error {
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
	defer delete(conn.ConnMgrs, groupId)

	connMgr.LeaveAllChannels()

	return nil
}

func (conn *Conn) GetConnMgr(groupId string) (*ConnMgr, error) {
	if connMgr, ok := conn.ConnMgrs[groupId]; ok {
		return connMgr, nil
	}
	return nil, fmt.Errorf("connMgr for group <%s> not exist", groupId)
}

func (connMgr *ConnMgr) InitGroupConnMgr(groupId string, ownerPubkey string, userSignPubkey string, cIface chaindef.ChainDataSyncIface) error {
	conn_log.Debugf("InitGroupConnMgr called, groupId <%s>", groupId)
	connMgr.UserChannelId = constants.USER_CHANNEL_PREFIX + groupId
	connMgr.ProducerChannelId = constants.PRODUCER_CHANNEL_PREFIX + groupId
	connMgr.GroupId = groupId
	connMgr.OwnerPubkey = ownerPubkey
	connMgr.UserSignPubkey = userSignPubkey
	connMgr.ProviderPeerIdPool = make(map[string]string)
	connMgr.ProducerPool = make(map[string]string)
	connMgr.PsConns = make(map[string]*pubsubconn.P2pPubSubConn)

	connMgr.DataHandlerIface = cIface

	//Rex
	if nodectx.GetNodeCtx().Node.RumExchange != nil {
		nodectx.GetNodeCtx().Node.RumExchange.ChainReg(connMgr.GroupId, cIface)
	}

	//initial ps conn for user channel
	connMgr.InitialPsConn()

	return nil
}

func (connMgr *ConnMgr) UpdProducers(pubkeys []string) error {
	conn_log.Debugf("UpdProducers, groupId <%s>", connMgr.GroupId)
	connMgr.ProducerPool = make(map[string]string)

	for _, pubkey := range pubkeys {
		connMgr.ProducerPool[pubkey] = pubkey
	}

	pk, _ := localcrypto.Libp2pPubkeyToEthBase64(connMgr.UserSignPubkey)
	if pk == "" {
		pk = connMgr.UserSignPubkey
	}

	if _, ok := connMgr.ProducerPool[pk]; ok {
		conn_log.Debugf("I am producer, create producer psconn, groupId <%s>", connMgr.GroupId)
		connMgr.StableProdPsConn = true
		connMgr.getProducerPsConn()
	} else {
		conn_log.Debugf("I am NOT producer, create producer psconn only when needed, groupId <%s>", connMgr.GroupId)
		connMgr.StableProdPsConn = false
	}

	return nil
}

func (connMgr *ConnMgr) LeaveAllChannels() error {
	conn_log.Debugf("LeaveChannel called, groupId <%s>", connMgr.GroupId)
	for channelId, psconn := range connMgr.PsConns {
		psconn.LeaveChannel()
		delete(connMgr.PsConns, channelId)
	}
	return nil
}

func (connMgr *ConnMgr) InitialPsConn() {
	conn_log.Debugf("<%s> InitialPsConn called", connMgr.GroupId)
	//userPsconn := nodectx.GetNodeCtx().Node.PubSubConnMgr.GetPubSubConnByChannelId(connMgr.UserChannelId, connMgr.DataHandlerIface)
	userPsconn := pubsubconn.GetPubSubConnByChannelId(context.Background(), nodectx.GetNodeCtx().Node.Pubsub, connMgr.UserChannelId, connMgr.DataHandlerIface, nodectx.GetNodeCtx().Node.NodeName)
	connMgr.PsConns[connMgr.UserChannelId] = userPsconn
}

func (connMgr *ConnMgr) getProducerPsConn() *pubsubconn.P2pPubSubConn {
	conn_log.Debugf("<%s> getProducerPsConn called", connMgr.GroupId)
	if psconn, ok := connMgr.PsConns[connMgr.ProducerChannelId]; ok {
		if !connMgr.StableProdPsConn { //is user, no need to keep producer psconn
			conn_log.Debugf("<%s> reset connection timer for producer psconn <%s>", connMgr.GroupId, connMgr.ProducerChannelId)
			connMgr.producerChannTimer.Stop()
			connMgr.producerChannTimer.Reset(CLOSE_PRD_CHANN_TIMER)
		} else {
			if connMgr.producerChannTimer != nil {
				conn_log.Debugf("<%s> stop producer psconn timer <%s>", connMgr.GroupId, connMgr.ProducerChannelId)
				connMgr.producerChannTimer.Stop()
			}
		}
		return psconn
	} else {
		//producerPsconn := nodectx.GetNodeCtx().Node.PubSubConnMgr.GetPubSubConnByChannelId(connMgr.ProducerChannelId, connMgr.DataHandlerIface)
		producerPsconn := pubsubconn.GetPubSubConnByChannelId(context.Background(), nodectx.GetNodeCtx().Node.Pubsub, connMgr.ProducerChannelId, connMgr.DataHandlerIface, nodectx.GetNodeCtx().Node.NodeName)
		connMgr.PsConns[connMgr.ProducerChannelId] = producerPsconn
		if !connMgr.StableProdPsConn {
			conn_log.Debugf("<%s> create close_conn timer for producer channel <%s>", connMgr.GroupId, connMgr.ProducerChannelId)
			connMgr.producerChannTimer = time.AfterFunc(CLOSE_PRD_CHANN_TIMER, func() {
				conn_log.Debugf("<%s> time up, close producer channel <%s>", connMgr.GroupId, connMgr.ProducerChannelId)

				psconn := connMgr.PsConns[connMgr.ProducerChannelId]
				psconn.LeaveChannel()
				//nodectx.GetNodeCtx().Node.PubSubConnMgr.LeaveChannel(connMgr.ProducerChannelId)
				delete(connMgr.PsConns, connMgr.ProducerChannelId)
			})
		}
		return producerPsconn
	}
}

func (connMgr *ConnMgr) getUserConn() *pubsubconn.P2pPubSubConn {
	//conn_log.Debugf("<%s> getUserConn called", connMgr.GroupId)
	return connMgr.PsConns[connMgr.UserChannelId]
}

func (connMgr *ConnMgr) SendTrxPubsub(trx *quorumpb.Trx, psChannel PsConnChanel, channelId ...string) error {
	conn_log.Debugf("<%s> SendTrxPubsub called", connMgr.GroupId)
	pkg := &quorumpb.Package{}

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
	}

	return fmt.Errorf("can not find psChannel")
}

func (connMgr *ConnMgr) SendReqTrxRex(trx *quorumpb.Trx) error {
	conn_log.Debugf("<%s> SendTrxRex called", connMgr.GroupId)
	if nodectx.GetNodeCtx().Node.RumExchange == nil {
		return errors.New("RumExchange is nil, please set enablerumexchange as true")
	}

	pbBytes, err := proto.Marshal(trx)
	if err != nil {
		return err
	}

	pkg := &quorumpb.Package{}
	pkg.Type = quorumpb.PackageType_TRX
	pkg.Data = pbBytes
	rummsg := &quorumpb.RumMsg{MsgType: quorumpb.RumMsgType_CHAIN_DATA, DataPackage: pkg}

	psconn := connMgr.getUserConn()
	channelpeers := psconn.Topic.ListPeers()
	return nodectx.GetNodeCtx().Node.RumExchange.Publish(trx.GroupId, channelpeers, rummsg)
}

func (connMgr *ConnMgr) SendRespTrxRex(trx *quorumpb.Trx, s network.Stream) error {
	conn_log.Debugf("<%s> SendRespTrxRex called", connMgr.GroupId)
	if nodectx.GetNodeCtx().Node.RumExchange == nil {
		return errors.New("RumExchange is nil, please set enablerumexchange as true")
	}

	if s == nil {
		return errors.New("Resp peer steam can't be nil")
	}

	pbBytes, err := proto.Marshal(trx)
	if err != nil {
		return err
	}
	pkg := &quorumpb.Package{}
	pkg.Type = quorumpb.PackageType_TRX
	pkg.Data = pbBytes
	rummsg := &quorumpb.RumMsg{MsgType: quorumpb.RumMsgType_CHAIN_DATA, DataPackage: pkg}
	//TODO:  add a timeout ctx to close the steam after timeout

	return nodectx.GetNodeCtx().Node.RumExchange.PublishToStream(rummsg, s) //publish to a stream
}

func (connMgr *ConnMgr) BroadcastHBMsg(hbb *quorumpb.HBMsgv1) error {
	pkg := &quorumpb.Package{}

	pbBytes, err := proto.Marshal(hbb)
	if err != nil {
		return err
	}

	pkg.Type = quorumpb.PackageType_HBB
	pkg.Data = pbBytes

	pkgBytes, err := proto.Marshal(pkg)
	if err != nil {
		return err
	}

	psconn := connMgr.getProducerPsConn()
	return psconn.Publish(pkgBytes)
}

func (connMgr *ConnMgr) BroadcastConsensusMsg(msg *quorumpb.ConsensusMsg) error {
	conn_log.Debugf("<%s> BroadcastConsensusMsg called", connMgr.GroupId)
	pkg := &quorumpb.Package{}

	pbBytes, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	pkg.Type = quorumpb.PackageType_CONSENSUS
	pkg.Data = pbBytes

	pkgBytes, err := proto.Marshal(pkg)
	if err != nil {
		return err
	}

	conn_log.Debugf("<%s> Send ConsensusReq via Producer_Channel", connMgr.GroupId)
	psconn := connMgr.getProducerPsConn()
	return psconn.Publish(pkgBytes)

}

func (connMgr *ConnMgr) BroadcastBlock(blk *quorumpb.Block) error {
	conn_log.Debugf("<%s> SendBlockPsconn called", connMgr.GroupId)
	pbBytes, err := proto.Marshal(blk)
	if err != nil {
		return err
	}

	pkg := &quorumpb.Package{}
	pkg.Type = quorumpb.PackageType_BLOCK
	pkg.Data = pbBytes
	pkgBytes, err := proto.Marshal(pkg)
	if err != nil {
		return err
	}

	conn_log.Debugf("<%s> Send block via User_Channel", connMgr.GroupId)
	psconn := connMgr.getUserConn()
	return psconn.Publish(pkgBytes)
}
