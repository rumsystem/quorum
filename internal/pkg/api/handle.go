package api

import (
	"context"
	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	"github.com/huo-ju/quorum/internal/pkg/p2p"
)

type (
	Handler struct {
		Ctx      context.Context
		Node     *p2p.Node
		ChainCtx *chain.ChainContext
	}
)
