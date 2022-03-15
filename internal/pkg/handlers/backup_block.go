package handlers

import (
	"fmt"
	"os"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/storage"
)

var (
	logger = logging.Logger("handlers")
)

// BackupBlock get block from data db and backup to `backupPath`
func BackupBlock(dataDir, peerName, backupPath string) {
	datapath := dataDir + "/" + peerName
	dbManager, err := storage.CreateDb(datapath)
	if err != nil {
		logger.Fatalf("storage.CreateDb failed: %s", err)
	}
	dbManager.TryMigration(0) //TOFIX: pass the node data_ver

	// backup block
	backupDB := storage.QSBadger{}
	os.RemoveAll(backupPath)
	err = backupDB.Init(backupPath)
	if err != nil {
		logger.Fatalf("backupDB.Init failed: %s", err)
	}

	// NOTE: hardcode
	key := "default" + "_" + storage.BLK_PREFIX + "_"
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
