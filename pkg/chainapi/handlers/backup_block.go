//go:build !js
// +build !js

package handlers

import (
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/storage"
)

var (
	logger   = logging.Logger("handlers")
	nodename = "default" // NOTE: hardcode
)

// BackupBlock get block from data db and backup to `backupPath`
func BackupBlock(dataDir, peerName, backupPath string) {
	datapath := dataDir + "/" + peerName
	dbManager, err := storage.CreateDb(datapath)
	if err != nil {
		logger.Fatalf("storage.CreateDb failed: %s", err)
	}
	defer dbManager.Db.Close()
	defer dbManager.GroupInfoDb.Close()

	// backup block
	backupDB := storage.QSBadger{}
	if err := backupDB.Init(backupPath); err != nil {
		logger.Fatalf("backupDB.Init failed: %s", err)
	}
	defer backupDB.Close()

	key := getBlockPrefixKey()
	err = dbManager.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}

		if err := backupDB.Set(k, v); err != nil {
			return fmt.Errorf("backupDB.Set failed: %s", err)
		}
		return nil
	})

	if err != nil {
		logger.Fatalf("dbManager.Db.PrefixForeach failed: %s", err)
	}
}
