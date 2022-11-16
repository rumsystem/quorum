//go:build !js
// +build !js

package handlers

import (
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/storage"
)

// BackupBlock get block from data db and backup to `backupPath`
func BackupBlock(dataDir, peerName, backupDataPath string) {
	datapath := dataDir + "/" + peerName
	dbManager, err := storage.CreateDb(datapath)
	if err != nil {
		logger.Fatalf("storage.CreateDb failed: %s", err)
	}
	defer dbManager.Db.Close()
	defer dbManager.GroupInfoDb.Close()

	// backup block
	backupDbMgr, err := storage.CreateDb(backupDataPath)
	if err != nil {
		logger.Fatalf("storage.CreateDb %s failed: %s", backupDataPath, err)
	}
	defer backupDbMgr.Db.Close()
	defer backupDbMgr.GroupInfoDb.Close()

	key := getBlockPrefixKey()
	err = dbManager.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}

		if err := backupDbMgr.Db.Set(k, v); err != nil {
			return fmt.Errorf("backupDbMgr.Db.Set failed: %s", err)
		}
		return nil
	})

	if err != nil {
		logger.Fatalf("backupDbMgr.Db.PrefixForeach failed: %s", err)
	}
}
