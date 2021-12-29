package pubsubconn

import (
	"context"
	logging "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var channel_log = logging.Logger("chan")

type P2pPubSubConn struct {
	Cid          string
	Topic        *pubsub.Topic
	Subscription *pubsub.Subscription
	chain        Chain
	ps           *pubsub.PubSub
	nodename     string
	Ctx          context.Context
}

type PubSubConnMgr struct {
	Ctx      context.Context
	ps       *pubsub.PubSub
	nodename string
	connmgr  map[string]*P2pPubSubConn
}

var pubsubconnmgr *PubSubConnMgr

func InitPubSubConnMgr(ctx context.Context, ps *pubsub.PubSub, nodename string) *PubSubConnMgr {
	if pubsubconnmgr == nil {
		connmap := map[string]*P2pPubSubConn{}
		pubsubconnmgr = &PubSubConnMgr{ctx, ps, nodename, connmap}
	}
	return pubsubconnmgr
}
func GetPubSubConnMgr() *PubSubConnMgr {
	return pubsubconnmgr
}

func (pscm *PubSubConnMgr) GetPubSubConnByChannelId(channelId string, chain Chain) *P2pPubSubConn {
	_, ok := pscm.connmgr[channelId]
	if ok == false {
		psconn := &P2pPubSubConn{Ctx: pscm.Ctx, ps: pscm.ps, nodename: pscm.nodename}
		if chain != nil {
			psconn.JoinChannel(channelId, chain)
		} else {
			psconn.JoinChannelAsExchange(channelId)
		}
		pscm.connmgr[channelId] = psconn
	}
	return pscm.connmgr[channelId]
}

func (pscm *PubSubConnMgr) LeaveChannel(channelId string) {
	_, ok := pscm.connmgr[channelId]
	if ok == true {
		psconn := pscm.connmgr[channelId]
		psconn.Subscription.Cancel()
		psconn.Topic.Close()
		delete(pscm.connmgr, channelId)
		channel_log.Infof("Leave channel <%s> done", channelId)
	} else {
		channel_log.Infof("psconn channel <%s> not exist", channelId)
	}

}

func InitP2pPubSubConn(ctx context.Context, ps *pubsub.PubSub, nodename string) *P2pPubSubConn {
	return &P2pPubSubConn{Ctx: ctx, ps: ps, nodename: nodename}
}

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

	go psconn.handleExchangeChannel()
	//TODO: add a timer to leave the exchange channel
	return nil
}

func (psconn *P2pPubSubConn) JoinChannel(cId string, chain Chain) error {
	psconn.Cid = cId
	psconn.chain = chain

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

	go psconn.handleGroupChannel()
	return nil
}

func (psconn *P2pPubSubConn) Publish(data []byte) error {
	return psconn.Topic.Publish(psconn.Ctx, data)
}

func (psconn *P2pPubSubConn) handleGroupChannel() error {
	for {
		msg, err := psconn.Subscription.Next(psconn.Ctx)
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
						psconn.chain.HandleBlock(blk)
					} else {
						channel_log.Warning(err.Error())
					}
				} else if pkg.Type == quorumpb.PackageType_TRX {
					var trx *quorumpb.Trx
					trx = &quorumpb.Trx{}
					err := proto.Unmarshal(pkg.Data, trx)
					if err == nil {
						psconn.chain.HandleTrx(trx)
					} else {
						channel_log.Warningf(err.Error())
					}
				}
			} else {
				channel_log.Warningf(err.Error())
				channel_log.Warningf("%s", msg.Data)
			}
		} else {
			channel_log.Errorf(err.Error())
			return err
		}
	}
}

func (psconn *P2pPubSubConn) handleExchangeChannel() error {
	for {
		msg, err := psconn.Subscription.Next(psconn.Ctx)
		if err == nil {
			channel_log.Infof("recv data: %s from channel: %s", msg.Data, psconn.Cid)
			//if string(msg.Data[:]) == "ping" {
			//	channel_log.Infof("recv normal msg and send pong resp: %s", msg.Data)
			//	psconn.Publish([]byte("pong"))
			//} else {
			//	channel_log.Infof("recv data: %s from channel: %s", msg.Data, psconn.Cid)
			//}
		} else {
			channel_log.Errorf(err.Error())
			return err
		}
	}
}
