package pubsubconn

import (
	"context"
	"fmt"
	"strings"

	pubsub "github.com/huo-ju/quercus/pkg/pubsub"
	"github.com/huo-ju/quercus/pkg/quality"
	iface "github.com/rumsystem/quorum/internal/pkg/chaindataciface"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var (
	PRODUCER_CHANNEL_PREFIX = "prod_channel_"
)

type QuercusConn struct {
	Cid          string
	Subscription *pubsub.Subscription
	chain        iface.ChainDataHandlerIface
	ps           *pubsub.Pubsub
	nodename     string
	Ctx          context.Context
}

var quercus_log = logging.Logger("chan")

type ChannelType int

const (
	ProducerChan ChannelType = iota
	UserChan
)

func InitQuercusConn(ctx context.Context, ps *pubsub.Pubsub, nodename string) *QuercusConn {
	return &QuercusConn{Ctx: ctx, ps: ps, nodename: nodename}
}

func (qconn *QuercusConn) JoinChannel(cId string, chain iface.ChainDataHandlerIface) error {
	qconn.Cid = cId
	qconn.chain = chain
	qconn.Subscription = qconn.ps.Subscribe(qconn.nodename, cId)
	quercus_log.Infof("Subscribe <%s> done", cId)
	fmt.Printf("Subscribe <%s> done\n", cId)

	chantype := UserChan
	if strings.HasPrefix(cId, PRODUCER_CHANNEL_PREFIX) {
		chantype = ProducerChan
	}
	go qconn.handleGroupChannel(chantype)
	return nil
}

func (qconn *QuercusConn) Publish(data []byte) error {
	fmt.Printf("Publish To:%s\n", qconn.Cid)
	qconn.ps.Publish(qconn.Cid, data)
	return nil
}

func (qconn *QuercusConn) handleGroupChannel(chantype ChannelType) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dqa := quality.NewDelayQualityAgent(0, 15)

	for {
		msg, err := qconn.Subscription.Next(ctx)
		if err == nil {

			msg := dqa.Pass(msg)
			data, ok := msg.([]byte)
			if ok == false {
				return fmt.Errorf("input msg error")
			}

			var pkg quorumpb.Package
			err = proto.Unmarshal(data, &pkg)

			if err == nil {
				if pkg.Type == quorumpb.PackageType_BLOCK {
					//is block
					var blk *quorumpb.Block
					blk = &quorumpb.Block{}
					err := proto.Unmarshal(pkg.Data, blk)
					if err == nil {
						qconn.chain.HandleBlockPsConn(blk)
					} else {
						channel_log.Warning(err.Error())
					}
				} else if pkg.Type == quorumpb.PackageType_TRX {
					var trx *quorumpb.Trx
					trx = &quorumpb.Trx{}
					err := proto.Unmarshal(pkg.Data, trx)
					if err == nil {
						qconn.chain.HandleTrxPsConn(trx)
					} else {
						quercus_log.Warningf(err.Error())
					}
				}
			}
			//fmt.Printf("Node:(%s) [%s] got pubsub msg\n", channel.Subscription.Id, channel.Cid)
		} else {
			fmt.Println(err)
		}
	}
}
