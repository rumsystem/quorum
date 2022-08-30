package pubsubconn

import (
	"context"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p-core/network"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	chaindef "github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/stats"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"

	"google.golang.org/protobuf/proto"
)

var channel_log = logging.Logger("chan")

type P2pPubSubConn struct {
	Cid          string
	Topic        *pubsub.Topic
	Subscription *pubsub.Subscription
	chain        chaindef.ChainDataSyncIface
	ps           *pubsub.PubSub
	nodename     string
	Ctx          context.Context
	mu           sync.RWMutex
	cancel       context.CancelFunc
	pubsubcancel pubsub.RelayCancelFunc
}

type PubSubConnMgr struct {
	Ctx      context.Context
	ps       *pubsub.PubSub
	nodename string
	connmgr  sync.Map
}

var pubsubconnmgr *PubSubConnMgr

func InitPubSubConnMgr(ctx context.Context, ps *pubsub.PubSub, nodename string) *PubSubConnMgr {
	if pubsubconnmgr == nil {
		pubsubconnmgr = &PubSubConnMgr{Ctx: ctx, ps: ps, nodename: nodename}
	}
	return pubsubconnmgr
}

func GetPubSubConnMgr() *PubSubConnMgr {
	return pubsubconnmgr
}

func (pscm *PubSubConnMgr) GetPubSubConnByChannelId(channelId string, cdhIface chaindef.ChainDataSyncIface) *P2pPubSubConn {
	_, ok := pscm.connmgr.Load(channelId)
	if ok == false {
		ctxwithcancel, cancel := context.WithCancel(pscm.Ctx)
		psconn := &P2pPubSubConn{Ctx: ctxwithcancel, cancel: cancel, ps: pscm.ps, nodename: pscm.nodename}
		if cdhIface != nil {
			psconn.JoinChannel(channelId, cdhIface)
		} else {
			// join channel as exchange
			psconn.JoinChannel(channelId, nil)
		}
		pscm.connmgr.Store(channelId, psconn)
	}
	psconn, _ := pscm.connmgr.Load(channelId)
	return psconn.(*P2pPubSubConn)
}

func (pscm *PubSubConnMgr) CreatePubSubRelayByChannelId(channelId string) *P2pPubSubConn {
	_, ok := pscm.connmgr.Load(channelId)
	if ok == false {
		ctxwithcancel, cancel := context.WithCancel(pscm.Ctx)
		psconn := &P2pPubSubConn{Ctx: ctxwithcancel, cancel: cancel, ps: pscm.ps, nodename: pscm.nodename}
		psconn.JoinChannelAsRelay(channelId)
		pscm.connmgr.Store(channelId, psconn)
	}
	psconn, _ := pscm.connmgr.Load(channelId)
	return psconn.(*P2pPubSubConn)
}

func (pscm *PubSubConnMgr) LeaveRelayChannel(channelId string) {
	psconni, ok := pscm.connmgr.Load(channelId)
	if ok == true {
		psconn := psconni.(*P2pPubSubConn)
		psconn.mu.Lock()
		defer psconn.mu.Unlock()
		if psconn.pubsubcancel != nil {
			psconn.pubsubcancel()
		}
	} else {
		channel_log.Infof("psconn relay channel <%s> not exist", channelId)
	}
}

func (pscm *PubSubConnMgr) LeaveChannel(channelId string) {
	psconni, ok := pscm.connmgr.Load(channelId)
	if ok == true {
		psconn := psconni.(*P2pPubSubConn)
		psconn.mu.Lock()
		defer psconn.mu.Unlock()
		if psconn.cancel != nil {
			psconn.cancel()
		}
		if psconn.Subscription != nil {
			psconn.Subscription.Cancel()
		}
		if psconn.Topic != nil {
			psconn.Topic.Close()
		}
		pscm.connmgr.Delete(channelId)
		channel_log.Infof("Leave channel <%s> done", channelId)
	} else {
		channel_log.Infof("psconn channel <%s> not exist", channelId)
	}

}

func (psconn *P2pPubSubConn) JoinChannelAsRelay(cId string) error {
	var err error
	psconn.Cid = cId
	psconn.Topic, err = psconn.ps.Join(cId)
	if err != nil {
		channel_log.Infof("Join <%s> failed", cId)
		return err
	} else {
		channel_log.Infof("Join <%s> done", cId)
	}
	relayCancel, err := psconn.Topic.Relay()
	psconn.pubsubcancel = func() {
		relayCancel()
		channel_log.Infof("Cancel relay <%s> done", cId)

	}
	return err
}

func (psconn *P2pPubSubConn) JoinChannel(cId string, cdhIface chaindef.ChainDataSyncIface) error {
	psconn.Cid = cId

	// cdhIface == nil, join channel as exchange
	if cdhIface != nil {
		psconn.chain = cdhIface
	}

	log := stats.NetworkStats{
		From:      stats.GetLocalPeerID(),
		Topic:     cId,
		Action:    stats.JoinTopic,
		Direction: network.DirOutbound,
		Size:      0,
		Success:   false,
	}

	var err error
	//TODO: share the ps
	psconn.Topic, err = psconn.ps.Join(cId)
	if err != nil {
		channel_log.Infof("Join <%s> failed", cId)
		if e := stats.GetStatsDB().AddNetworkLog(&log); e != nil {
			channel_log.Warningf("add network log to db failed: %s", e)
		}
		return err
	} else {
		channel_log.Infof("Join <%s> done", cId)
		log.Success = true
		if e := stats.GetStatsDB().AddNetworkLog(&log); e != nil {
			channel_log.Warningf("add network log to db failed: %s", e)
		}
	}

	log = stats.NetworkStats{
		From:      stats.GetLocalPeerID(),
		Topic:     cId,
		Action:    stats.SubscribeTopic,
		Direction: network.DirOutbound,
		Size:      0,
		Success:   false,
	}

	psconn.Subscription, err = psconn.Topic.Subscribe()
	if err != nil {
		channel_log.Fatalf("Subscribe <%s> failed: %s", cId, err)
		if e := stats.GetStatsDB().AddNetworkLog(&log); e != nil {
			channel_log.Warningf("add network log to db failed: %s", e)
		}
		return err
	} else {
		channel_log.Infof("Subscribe <%s> done", cId)
		log.Success = true
		if e := stats.GetStatsDB().AddNetworkLog(&log); e != nil {
			channel_log.Warningf("add network log to db failed: %s", e)
		}
	}

	go psconn.handleGroupChannel(psconn.Ctx)
	return nil
}

func (psconn *P2pPubSubConn) Publish(data []byte) error {
	psconn.mu.Lock()
	defer psconn.mu.Unlock()
	if psconn.Topic == nil {
		return fmt.Errorf("Topic has been closed.")
	}

	err := psconn.Topic.Publish(psconn.Ctx, data)

	success := err == nil
	log := stats.NetworkStats{
		From:      stats.GetLocalPeerID(),
		Topic:     psconn.Topic.String(),
		Action:    stats.PublishToTopic,
		Direction: network.DirOutbound,
		Size:      stats.GetBinarySize(data),
		Success:   success,
	}
	if e := stats.GetStatsDB().AddNetworkLog(&log); e != nil {
		channel_log.Warningf("add network log to db failed: %s", err)
	}

	return err
}

func (psconn *P2pPubSubConn) handleGroupChannel(ctx context.Context) error {
	for {
		msg, err := psconn.Subscription.Next(ctx)
		if err == nil {
			var pkg quorumpb.Package
			if err := proto.Unmarshal(msg.Data, &pkg); err == nil {
				log := stats.NetworkStats{
					To:        stats.GetLocalPeerID(),
					Topic:     *msg.Topic,
					Action:    stats.ReceiveFromTopic,
					Direction: network.DirInbound,
					Size:      stats.GetProtoSize(&pkg),
					Success:   true,
				}
				if err := stats.GetStatsDB().AddNetworkLog(&log); err != nil {
					channel_log.Warningf("add network log to db failed: %s", err)
				}

				if pkg.Type == quorumpb.PackageType_BLOCK {
					//is block
					var blk *quorumpb.Block
					blk = &quorumpb.Block{}
					err := proto.Unmarshal(pkg.Data, blk)
					if err == nil {
						psconn.chain.HandleBlockPsConn(blk)
					} else {
						channel_log.Warning(err.Error())
					}
				} else if pkg.Type == quorumpb.PackageType_TRX {
					var trx *quorumpb.Trx
					trx = &quorumpb.Trx{}
					err := proto.Unmarshal(pkg.Data, trx)

					if err == nil {
						psconn.chain.HandleTrxPsConn(trx)
					} else {
						channel_log.Warningf(err.Error())
					}
				} else if pkg.Type == quorumpb.PackageType_SNAPSHOT {
					var snapshot *quorumpb.Snapshot
					snapshot = &quorumpb.Snapshot{}
					err := proto.Unmarshal(pkg.Data, snapshot)
					if err == nil {
						psconn.chain.HandleSnapshotPsConn(snapshot)
					} else {
						channel_log.Warningf(err.Error())
					}
				}
			} else {
				channel_log.Warningf(err.Error())
				channel_log.Warningf("%s", msg.Data)
			}
		} else {
			channel_log.Debugf(err.Error())
			return err
		}
	}
}
