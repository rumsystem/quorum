package conn

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/network"
	chaindef "github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/internal/pkg/conn/pubsubconn"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/constants"
	"github.com/rumsystem/quorum/pkg/data"
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
	DataHandlerIface   chaindef.ChainDataSyncIface
	//TODO: sync.map
	//ps *pubsub.PubSub

	pscounsmu sync.RWMutex
	PsConns   map[string]*pubsubconn.P2pPubSubConn // key: channelId
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

func (connMgr *ConnMgr) LeaveAllChannels() error {
	conn_log.Debugf("LeaveChannel called, groupId <%s>", connMgr.GroupId)
	connMgr.pscounsmu.Lock()
	defer connMgr.pscounsmu.Unlock()
	for channelId, psconn := range connMgr.PsConns {
		psconn.LeaveChannel()
		delete(connMgr.PsConns, channelId)
	}
	return nil
}

func (connMgr *ConnMgr) InitialPsConn() {
	conn_log.Debugf("<%s> InitialPsConn called", connMgr.GroupId)
	connMgr.pscounsmu.Lock()
	defer connMgr.pscounsmu.Unlock()
	userPsconn := pubsubconn.GetPubSubConnByChannelId(context.Background(), nodectx.GetNodeCtx().Node.Pubsub, connMgr.UserChannelId, connMgr.DataHandlerIface, nodectx.GetNodeCtx().Node.NodeName)
	connMgr.PsConns[connMgr.UserChannelId] = userPsconn
}

func (connMgr *ConnMgr) getUserConn() *pubsubconn.P2pPubSubConn {
	//conn_log.Debugf("<%s> getUserConn called", connMgr.GroupId)
	return connMgr.PsConns[connMgr.UserChannelId]
}

func (connMgr *ConnMgr) SendUserTrxPubsub(trx *quorumpb.Trx, channelId ...string) error {
	conn_log.Debugf("<%s> SendTrxPubsub called", connMgr.GroupId)

	// check trx.Data size
	if _, err := data.IsTrxDataWithinSizeLimit(trx.Data); err != nil {
		return err
	}

	// compress trx.Data
	compressedContent := new(bytes.Buffer)
	if err := utils.Compress(bytes.NewReader(trx.Data), compressedContent); err != nil {
		return err
	}
	trx.Data = compressedContent.Bytes()

	pbBytes, err := proto.Marshal(trx)
	if err != nil {
		return err
	}

	pkg := &quorumpb.Package{
		Type: quorumpb.PackageType_TRX,
		Data: pbBytes,
	}

	pkgBytes, err := proto.Marshal(pkg)
	if err != nil {
		return err
	}

	//conn_log.Debugf("<%s> Send trx via User_Channel", connMgr.GroupId)
	psconn := connMgr.getUserConn()
	return psconn.Publish(pkgBytes)
}

func (connMgr *ConnMgr) SendSyncReqMsgRex(req *quorumpb.ReqBlock) error {
	conn_log.Debugf("<%s> SendSyncReqMsgRex called", connMgr.GroupId)
	if nodectx.GetNodeCtx().Node.RumExchange == nil {
		return errors.New("RumExchange is nil, please set enablerumexchange as true")
	}

	//create sync msg
	pbBytes, err := proto.Marshal(req)
	if err != nil {
		return err
	}

	syncMsg := &quorumpb.SyncMsg{
		Type: quorumpb.SyncMsgType_REQ_BLOCK,
		Data: pbBytes,
	}

	//marshal sync msg
	syncMsgBytes, err := proto.Marshal(syncMsg)
	if err != nil {
		return err
	}

	//compress
	compressedContent := new(bytes.Buffer)
	if err := utils.Compress(bytes.NewReader(syncMsgBytes), compressedContent); err != nil {
		return err
	}

	//create pkg
	pkg := &quorumpb.Package{
		Type: quorumpb.PackageType_SYNC,
		Data: compressedContent.Bytes(),
	}

	rummsg := &quorumpb.RumDataMsg{MsgType: quorumpb.RumDataMsgType_CHAIN_DATA, DataPackage: pkg}

	psconn := connMgr.getUserConn()
	if psconn == nil {
		return fmt.Errorf("no user conn for %s. (can be ignored)", connMgr.GroupId)
	}
	channelpeers := psconn.Topic.ListPeers()
	return nodectx.GetNodeCtx().Node.RumExchange.Publish(req.GroupId, channelpeers, rummsg)
}

func (connMgr *ConnMgr) SendSyncRespMsgRex(resp *quorumpb.ReqBlockResp, s network.Stream) error {
	conn_log.Debugf("<%s> SendSyncRespMsgRex called", connMgr.GroupId)
	if nodectx.GetNodeCtx().Node.RumExchange == nil {
		return errors.New("RumExchange is nil, please set enablerumexchange as true")
	}

	if s == nil {
		return errors.New("resp peer steam can't be nil")
	}

	//create sync msg
	pbBytes, err := proto.Marshal(resp)
	if err != nil {
		return err
	}

	syncMsg := &quorumpb.SyncMsg{
		Type: quorumpb.SyncMsgType_REQ_BLOCK_RESP,
		Data: pbBytes,
	}

	//marshal sync msg
	syncMsgBytes, err := proto.Marshal(syncMsg)
	if err != nil {
		return err
	}

	//compress
	compressedContent := new(bytes.Buffer)
	if err := utils.Compress(bytes.NewReader(syncMsgBytes), compressedContent); err != nil {
		return err
	}

	//create pkg
	pkg := &quorumpb.Package{
		Type: quorumpb.PackageType_SYNC,
		Data: compressedContent.Bytes(),
	}

	rummsg := &quorumpb.RumDataMsg{MsgType: quorumpb.RumDataMsgType_CHAIN_DATA, DataPackage: pkg}
	return nodectx.GetNodeCtx().Node.RumExchange.PublishToStream(rummsg, s) //publish to a stream
}

func (connMgr *ConnMgr) BroadcastBlock(blk *quorumpb.Block) error {
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

	psconn := connMgr.getUserConn()
	return psconn.Publish(pkgBytes)
}

func (connMgr *ConnMgr) BroadcastBftMsg(msg *quorumpb.BftMsg) error {
	pkg := &quorumpb.Package{}

	pbBytes, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	pkg.Type = quorumpb.PackageType_BFT
	pkg.Data = pbBytes

	pkgBytes, err := proto.Marshal(pkg)
	if err != nil {
		return err
	}

	psconn := connMgr.getUserConn()
	return psconn.Publish(pkgBytes)
}
