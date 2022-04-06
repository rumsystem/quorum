package pubsubconn

import (
	"context"
	"fmt"
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	iface "github.com/rumsystem/quorum/internal/pkg/chaindataciface"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var channel_log = logging.Logger("chan")

type P2pPubSubConn struct {
	Cid          string
	Topic        *pubsub.Topic
	Subscription *pubsub.Subscription
	chain        iface.ChainDataHandlerIface
	ps           *pubsub.PubSub
	nodename     string
	Ctx          context.Context
	mu           sync.RWMutex
	cancel       context.CancelFunc
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

func (pscm *PubSubConnMgr) GetPubSubConnByChannelId(channelId string, cdhIface iface.ChainDataHandlerIface) *P2pPubSubConn {
	_, ok := pscm.connmgr.Load(channelId)
	if ok == false {
		ctxwithcancel, cancel := context.WithCancel(pscm.Ctx)
		psconn := &P2pPubSubConn{Ctx: ctxwithcancel, cancel: cancel, ps: pscm.ps, nodename: pscm.nodename}
		if cdhIface != nil {
			psconn.JoinChannel(channelId, cdhIface)
		} else {
			psconn.JoinChannelAsExchange(channelId)
		}
		pscm.connmgr.Store(channelId, psconn)
	}
	psconn, _ := pscm.connmgr.Load(channelId)
	return psconn.(*P2pPubSubConn)
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

//func InitP2pPubSubConn(ctx context.Context, ps *pubsub.PubSub, nodename string) *P2pPubSubConn {
//	ctxwithcancel, cancel := context.WithCancel(ctx)
//	return &P2pPubSubConn{Ctx: ctxwithcancel, cancel: cancel, ps: ps, nodename: nodename}
//}

func (psconn *P2pPubSubConn) JoinChannelAsExchange(cId string) error {
	var err error
	psconn.Cid = cId
	psconn.Topic, err = psconn.ps.Join(cId)
	if err != nil {
		channel_log.Errorf("Join <%s> failed", cId)
		return err
	} else {
		channel_log.Errorf("Join <%s> done", cId)
	}

	psconn.Subscription, err = psconn.Topic.Subscribe()
	if err != nil {
		channel_log.Errorf("Subscribe <%s> failed", cId)
		channel_log.Errorf(err.Error())
		return err
	} else {
		channel_log.Infof("Subscribe <%s> done", cId)
	}

	//TODO: add a timer to leave the exchange channel
	return nil
}

func (psconn *P2pPubSubConn) JoinChannel(cId string, cdhIface iface.ChainDataHandlerIface) error {
	psconn.Cid = cId
	psconn.chain = cdhIface

	var err error
	//TODO: share the ps
	psconn.Topic, err = psconn.ps.Join(cId)
	if err != nil {
		channel_log.Infof("Join <%s> failed", cId)
		return err
	} else {
		channel_log.Infof("Join <%s> done", cId)
	}

	psconn.Subscription, err = psconn.Topic.Subscribe()
	if err != nil {
		channel_log.Fatalf("Subscribe <%s> failed", cId)
		channel_log.Fatalf(err.Error())
		return err
	} else {
		channel_log.Infof("Subscribe <%s> done", cId)
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
	return psconn.Topic.Publish(psconn.Ctx, data)
}

func (psconn *P2pPubSubConn) handleGroupChannel(ctx context.Context) error {
	for {
		msg, err := psconn.Subscription.Next(ctx)
		if err == nil {
			var pkg quorumpb.Package
			err = proto.Unmarshal(msg.Data, &pkg)
			if err == nil {
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
