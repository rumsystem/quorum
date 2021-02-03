package api

import (
	"context"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type (
	Handler struct {
        Ctx context.Context
        PubsubTopic *pubsub.Topic
	}
)

