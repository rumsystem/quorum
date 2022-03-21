//go:build !js
// +build !js

package appdata

import "github.com/rumsystem/quorum/internal/pkg/storage"

func CreateAppDb(path string) (*AppDb, error) {
	var err error
	db := storage.QSBadger{}
	err = db.Init(path + "_appdb")
	if err != nil {
		return nil, err
	}

	app := NewAppDb()
	app.Db = &db
	app.DataPath = path
	return app, nil
}
