package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
)

type RestoreParam struct {
	Password    string `json:"password" validate:"required"`
	BackupFile  string `json:"backup_file" validate:"required"`
	KeystoreDir string `json:"keystore_dir" validate:"required"`
	ConfigDir   string `json:"config_dir" validate:"required"`
	SeedDir     string `json:"seed_dir" validate:"required"`
}

// Restore restores a backup file to given directories.
func Restore(params RestoreParam) error {
	content, err := ioutil.ReadFile(params.BackupFile)
	if err != nil {
		return fmt.Errorf("Failed to read backup file: %s", err)
	}

	var backup BackupResult
	if err := json.Unmarshal(content, &backup); err != nil {
		return fmt.Errorf("Failed to unmarshal backup file: %s", err)
	}

	err = localcrypto.Restore(
		params.Password, backup.Seeds, backup.Keystore, backup.Config,
		params.SeedDir, params.KeystoreDir, params.ConfigDir,
	)
	if err != nil {
		return fmt.Errorf("Failed to restore: %s", err)
	}

	return nil
}
