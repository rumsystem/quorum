package api

import (
	"context"

	"github.com/rumsystem/quorum/internal/pkg/appdata"
	"github.com/rumsystem/quorum/internal/pkg/storage"
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
