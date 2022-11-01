//go:build !js
// +build !js

package appdata

import (
	"context"

	"github.com/rumsystem/quorum/internal/pkg/storage"
)

func CreateAppDb(path string) (*AppDb, error) {
	ctx := context.Background()
	db, err := storage.NewStore(ctx, path, "appdb")
	if err != nil {
		return nil, err
	}

	app := NewAppDb()
	app.Db = db
	app.DataPath = path
	return app, nil
}
