package api

import (
	"context"

	"github.com/rumsystem/quorum/internal/pkg/appdata"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
)

type (
	Handler struct {
		Ctx        context.Context
		Node       *p2p.Node
		NodeCtx    *nodectx.NodeCtx
		GitCommit  string
		Appdb      *appdata.AppDb
		ChainAPIdb def.APIHandlerIface
	}
)
