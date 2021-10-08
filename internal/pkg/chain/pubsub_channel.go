package chain

import (
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	logging "github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"google.golang.org/protobuf/proto"
)

var channel_log = logging.Logger("chan")

type PubsubChannel struct {
	Cid          string
	Topic        *pubsub.Topic
	Subscription *pubsub.Subscription
	chain        *Chain
}

func (channel *PubsubChannel) JoinChannel(cId string, chain *Chain) error {

	channel.Cid = cId
	channel.chain = chain

	var err error
	channel.Topic, err = GetNodeCtx().node.Pubsub.Join(cId)
	if err != nil {
		channel_log.Infof("Join <%s> failed", cId)
		return err
	} else {
		channel_log.Infof("Join <%s> done", cId)
	}

	channel.Subscription, err = channel.Topic.Subscribe()
	if err != nil {
		channel_log.Fatalf("Subscribe <%s> failed", cId)
		channel_log.Fatalf(err.Error())
		return err
	} else {
		channel_log.Infof("Subscribe <%s> done", cId)
	}

	go channel.handleGroupChannel()

	return nil
}

func (channel *PubsubChannel) Publish(data []byte) error {
	return channel.Topic.Publish(GetNodeCtx().Ctx, data)
}

func (channel *PubsubChannel) handleGroupChannel() error {
	for {
		msg, err := channel.Subscription.Next(GetNodeCtx().Ctx)
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
						channel.chain.HandleBlock(blk)
					} else {
						chain_log.Warning(err.Error())
					}
				} else if pkg.Type == quorumpb.PackageType_TRX {
					var trx *quorumpb.Trx
					trx = &quorumpb.Trx{}
					err := proto.Unmarshal(pkg.Data, trx)
					if err == nil {
						if trx.Version != GetNodeCtx().Version {
							channel_log.Infof("Version mismatch")
						} else {
							channel.chain.HandleTrx(trx)
						}
					} else {
						channel_log.Warningf(err.Error())
					}
				}
			} else {
				channel_log.Warningf(err.Error())
			}
		} else {
			channel_log.Fatalf(err.Error())
			return err
		}
	}
}
