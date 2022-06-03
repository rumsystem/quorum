package nodesdkapi

import (
	"context"

	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type (
	NodeSDKHandler struct {
		Ctx        context.Context
		NodeSdkCtx *nodesdkctx.NodeSdkCtx
		GitCommit  string
	}
)

type ErrorResponse struct {
	Error string `json:"error" validate:"required"`
}
