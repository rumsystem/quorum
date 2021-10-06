package api

import (
	"context"

	"github.com/huo-ju/quorum/internal/pkg/appdata"
	"github.com/huo-ju/quorum/internal/pkg/storage"
)

type Handler struct {
	Ctx       context.Context
	Appdb     *appdata.AppDb
	Chaindb   *storage.DbMgr
	Apiroot   string
	GitCommit string
	ConfigDir string
	PeerName  string
	NodeName  string
}
