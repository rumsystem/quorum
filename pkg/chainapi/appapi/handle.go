package appapi

import (
	"context"

	"github.com/rumsystem/quorum/internal/pkg/appdata"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
)

type Handler struct {
	Ctx       context.Context
	Appdb     *appdata.AppDb
	Trxdb     def.TrxStorageIface
	Apiroot   string
	GitCommit string
	ConfigDir string
	PeerName  string
	NodeName  string
}
