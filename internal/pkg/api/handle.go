package api

import (
	"context"
	"github.com/huo-ju/quorum/internal/pkg/p2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type (
	Handler struct {
		Ctx         context.Context
		Node        *p2p.Node
		PubsubTopic *pubsub.Topic
	}
)
