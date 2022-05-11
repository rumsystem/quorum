package handlers

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p"
	"github.com/rumsystem/quorum/internal/pkg/logging"
)

type PingResp struct {
	TTL [10]int64 `json:"ttl"`
}

// pubsub ping
func Ping(ps *pubsub.PubSub, id peer.ID, remote string) (*PingResp, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	psping := p2p.NewPSPingService(ctx, ps, id)
	res, err := psping.PingReq(remote)
	if err != nil {
		return nil, err
	}

	return &PingResp{res}, nil
}

type P2PPingResp struct {
	TTL [10]int64 `json:"ttl"`
}

var pingLogger = logging.Logger("ping")

// p2p ping
func P2PPing(h host.Host, remote string) (*P2PPingResp, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ps := p2p.NewPingService(h)

	resp := &P2PPingResp{}

	for _, pid := range h.Peerstore().Peers() {
		if pid.String() == remote {
			ch := ps.Ping(ctx, pid)

			for i := 0; i < 10; i++ {
				res := <-ch
				if res.Error != nil {
					resp.TTL[i] = 0
					pingLogger.Error(res.Error.Error())
				} else {
					resp.TTL[i] = res.RTT.Milliseconds()
				}
			}
			return resp, nil
		}
	}

	return nil, fmt.Errorf("peer not found")
}
