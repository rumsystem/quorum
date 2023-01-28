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
	quorumpb "github.com/rumsystem/quorum/pkg/pb"

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
}

func GetPubSubConnByChannelId(ctx context.Context, ps *pubsub.PubSub, channelId string, cdhIface chaindef.ChainDataSyncIface, nodename string) *P2pPubSubConn {
	ctxwithcancel, cancel := context.WithCancel(ctx)
	psconn := &P2pPubSubConn{Ctx: ctxwithcancel, cancel: cancel, ps: ps, nodename: nodename}
	if cdhIface != nil {
		psconn.JoinChannel(channelId, cdhIface)
	} else {
		// join channel as exchange
		psconn.JoinChannel(channelId, nil)
	}
	return psconn
}

func (psconn *P2pPubSubConn) LeaveChannel() {
	channelId := psconn.Cid
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
	channel_log.Infof("Leave channel <%s> done", channelId)
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
				psconn.chain.HandlePsConnMessage(&pkg)

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
