// go:build js && wasm
//go:build js && wasm
// +build js,wasm

package stats

import (
	"github.com/rumsystem/quorum/internal/pkg/storage"
)

func openStatsDB(path string) (storage.QuorumStorage, error) {
	db := storage.QSIndexDB{}
	if err := db.Init(path + dbPathSuffix); err != nil {
		return nil, err
	}
	return &db, nil
}
