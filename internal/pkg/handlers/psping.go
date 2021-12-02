package handlers

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/rumsystem/quorum/internal/pkg/p2p"
)

type PingResp struct {
	TTL [10]int64 `json:"ttl"`
}

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
