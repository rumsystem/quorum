package pubsubconn

import (
	"context"
	"fmt"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	chaindef "github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/metric"
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

	var err error
	//TODO: share the ps
	psconn.Topic, err = psconn.ps.Join(cId)
	if err != nil {
		channel_log.Infof("Join <%s> failed", cId)
		metric.FailedCount.WithLabelValues(metric.ActionType.JoinTopic).Inc()
		return err
	} else {
		channel_log.Infof("Join <%s> done", cId)
		metric.SuccessCount.WithLabelValues(metric.ActionType.JoinTopic).Inc()
	}

	psconn.Subscription, err = psconn.Topic.Subscribe()
	if err != nil {
		channel_log.Errorf("Subscribe <%s> failed: %s", cId, err)
		metric.FailedCount.WithLabelValues(metric.ActionType.SubscribeTopic).Inc()
		return err
	} else {
		channel_log.Infof("Subscribe <%s> done", cId)
		metric.SuccessCount.WithLabelValues(metric.ActionType.SubscribeTopic).Inc()
	}

	go psconn.handleGroupChannel(psconn.Ctx)
	return nil
}

func (psconn *P2pPubSubConn) Publish(data []byte) error {
	publishctx, cancel := context.WithTimeout(psconn.Ctx, 2*time.Second)
	psconn.mu.Lock()
	defer psconn.mu.Unlock()
	defer cancel()
	if psconn.Topic == nil {
		return fmt.Errorf("Topic has been closed.")
	}

	//set a 2 Second timeout for pubsub Publish
	err := psconn.Topic.Publish(publishctx, data)
	if err != nil {
		metric.FailedCount.WithLabelValues(metric.ActionType.PublishToTopic).Inc()
	} else {
		size := float64(metric.GetBinarySize(data))
		metric.SuccessCount.WithLabelValues(metric.ActionType.PublishToTopic).Inc()
		metric.OutBytes.WithLabelValues(metric.ActionType.PublishToTopic).Set(size)
		metric.OutBytesTotal.WithLabelValues(metric.ActionType.PublishToTopic).Add(size)
	}

	return err
}

func (psconn *P2pPubSubConn) handleGroupChannel(ctx context.Context) error {
	for {
		msg, err := psconn.Subscription.Next(ctx)
		if err == nil {
			var pkg quorumpb.Package
			if err := proto.Unmarshal(msg.Data, &pkg); err == nil {
				size := float64(metric.GetProtoSize(&pkg))
				metric.SuccessCount.WithLabelValues(metric.ActionType.ReceiveFromTopic).Inc()
				metric.InBytes.WithLabelValues(metric.ActionType.ReceiveFromTopic).Set(size)
				metric.InBytesTotal.WithLabelValues(metric.ActionType.ReceiveFromTopic).Add(size)

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
				metric.FailedCount.WithLabelValues(metric.ActionType.ReceiveFromTopic).Inc()
				channel_log.Warningf(err.Error())
				channel_log.Warningf("%s", msg.Data)
			}
		} else {
			channel_log.Debugf(err.Error())
			return err
		}
	}
}
