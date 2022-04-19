package api

import (
	"context"

	"github.com/rumsystem/quorum/internal/pkg/appdata"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p"
	"github.com/rumsystem/quorum/internal/pkg/molassctx/nodectx"
)

type (
	Handler struct {
		Ctx       context.Context
		Node      *p2p.Node
		NodeCtx   *nodectx.NodeCtx
		GitCommit string
		Appdb     *appdata.AppDb
	}
)

type ErrorResponse struct {
	Error string `json:"error" validate:"required"`
}
