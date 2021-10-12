package api

import (
	"context"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/p2p"
)

type (
	Handler struct {
		Ctx       context.Context
		Node      *p2p.Node
		NodeCtx   *nodectx.NodeCtx
		GitCommit string
	}
)
