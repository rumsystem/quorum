package api

import (
	"context"
	"github.com/huo-ju/quorum/internal/pkg/appdata"
)

type Handler struct {
	Ctx       context.Context
	Appdb     *appdata.AppDb
	Apiroot   string
	GitCommit string
}
