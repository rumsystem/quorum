package api

import (
	"context"

	"github.com/huo-ju/quorum/internal/pkg/nodectx"
	"github.com/huo-ju/quorum/internal/pkg/p2p"
)

type (
	Handler struct {
		Ctx       context.Context
		Node      *p2p.Node
		NodeCtx   *nodectx.NodeCtx
		GitCommit string
	}
)
