package api

import (
	"context"

	"github.com/huo-ju/quorum/internal/pkg/appdata"
	"github.com/huo-ju/quorum/internal/pkg/chain"
)

type Handler struct {
	Ctx       context.Context
	Appdb     *appdata.AppDb
	Chaindb   *chain.DbMgr
	Apiroot   string
	GitCommit string
	ConfigDir string
	PeerName  string
	NodeName  string
}
