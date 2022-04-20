//go:build !js
// +build !js

package stats

import (
	"github.com/rumsystem/quorum/internal/pkg/storage"
)

func openStatsDB(path string) (storage.QuorumStorage, error) {
	db := storage.QSBadger{}
	if err := db.Init(path + dbPathSuffix); err != nil {
		return nil, err
	}
	return &db, nil
}
