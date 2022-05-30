//go:build !js
// +build !js

package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"filippo.io/age"
	"github.com/rumsystem/quorum/internal/pkg/appdata"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"

	"github.com/rumsystem/keystore/pkg/crypto"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
)

func GetDataPath(dataDir, peerName string) string {
	return filepath.Join(dataDir, peerName)
}

func getSeedBackupPath(dstPath string) string {
	return filepath.Join(dstPath, "seeds")
}

func getBlockBackupPath(dstPath string) string {
	return filepath.Join(dstPath, "block_db")
}

func getBlockRestorePath(peerName, dstPath string) string {
	dirName := fmt.Sprintf("%s_db", peerName)
	return filepath.Join(dstPath, dirName)
}

func getConfigBackupPath(dstPath string) string {
	return filepath.Join(dstPath, "config")
}

func getKeystoreBackupPath(dstPath string) string {
	return filepath.Join(dstPath, "keystore")
}

// for wasm side
func getWasmBackupPath(dstPath string) string {
	return filepath.Join(dstPath, "wasm/keystore")
}

func getBlockPrefixKey() string {
	return nodename + "_" + storage.BLK_PREFIX + "_"
}

// Backup backup block from data db and {config,keystore,seeds} directory
func Backup(config cli.Config, dstPath string, password string) {
	// get keystore password
	password, err := GetKeystorePassword(password)
	if err != nil {
		logger.Fatalf("handlers.GetKeystorePassword failed: %s", err)
	}

	// check keystore signature and encrypt
	if err := CheckSignAndEncryptWithKeystore(config.KeyStoreName, config.KeyStoreDir, config.ConfigDir, config.PeerName, password); err != nil {
		logger.Fatalf("check keystore failed: %s", err)
	}

	// check dst path
	if utils.DirExist(dstPath) || utils.FileExist(dstPath) {
		logger.Fatalf("backup directory %s is exists", dstPath)
	}

	dstPath, err = filepath.Abs(dstPath)
	if err != nil {
		logger.Fatalf("get abs path for %s failed: %s", dstPath, err)
	}

	// backup config directory
	configDstPath := getConfigBackupPath(dstPath)
	if err := utils.Copy(config.ConfigDir, configDstPath); err != nil {
		logger.Fatalf("copy %s => %s failed: %s", config.ConfigDir, dstPath, err)
	}

	// backup keystore
	keystoreDstPath := getKeystoreBackupPath(dstPath)
	if err := utils.Copy(config.KeyStoreDir, keystoreDstPath); err != nil {
		logger.Fatalf("copy %s => %s failed: %s", config.KeyStoreDir, dstPath, err)
	}

	/*
	   wasm need a single file in following format

	   ```
	   {"key": "", "value": ""}
	   {"key": "", "value": ""}
	   ...
	   ```
	   each row is encrypted(aes) then encoded with base64 algorithm
	*/
	wasmDstPath := getWasmBackupPath(dstPath)
	wasmKeystoreContent := ""
	if err := filepath.Walk(config.KeyStoreDir, func(path string, info os.FileInfo, err error) error {
		r, err := age.NewScryptRecipient(password)
		if err != nil {
			return err
		}

		keyBytes, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		pair := make(map[string]interface{})
		key := filepath.Base(path)
		pair["key"] = key
		pair["value"] = string(keyBytes)
		kvBytes, err := json.Marshal(pair)
		if err != nil {
			return err
		}

		output := new(bytes.Buffer)
		if err := crypto.AgeEncrypt([]age.Recipient{r}, bytes.NewReader(kvBytes), output); err != nil {
			return err
		}
		encryptedKvBytes, err := ioutil.ReadAll(output)
		if err != nil {
			return err
		}
		res := base64.StdEncoding.EncodeToString(encryptedKvBytes)
		wasmKeystoreContent += res
		return nil
	}); err != nil {
		logger.Fatalf("export keystore to wasm failed: %s", err)
	}
	if wasmKeystoreContent != "" {
		if err := ioutil.WriteFile(wasmDstPath, []byte(wasmKeystoreContent), 0644); err != nil {
			logger.Fatalf("export keystore to wasm failed: %s", err)
		}
	}

	// SaveAllGroupSeeds
	dataPath := GetDataPath(config.DataDir, config.PeerName)
	appdb, err := appdata.CreateAppDb(dataPath)
	if err != nil {
		logger.Fatalf("appdata.CreateAppDb failed: %s", err)
	}
	seedDstPath := getSeedBackupPath(dstPath)
	SaveAllGroupSeeds(appdb, seedDstPath)

	// backup block
	blockDstPath := getBlockBackupPath(dstPath)
	BackupBlock(config.DataDir, config.PeerName, blockDstPath)

	// zip backup directory
	zipFilePath := fmt.Sprintf("%s.zip", dstPath)
	defer utils.RemoveAll(dstPath)
	defer utils.RemoveAll(zipFilePath)
	if err := utils.ZipDir(dstPath, zipFilePath); err != nil {
		logger.Fatalf("utils.ZipDir(%s, %s) failed: %s", dstPath, zipFilePath, err)
	}

	// check keystore signature and encrypt
	if err := CheckSignAndEncryptWithKeystore(config.KeyStoreName, keystoreDstPath, configDstPath, config.PeerName, password); err != nil {
		logger.Fatalf("check keystore failed: %s", err)
	}

	if err := loadAndDecryptTrx(blockDstPath, seedDstPath); err != nil {
		logger.Fatalf("check backuped block data failed: %s", err)
	}

	// encrypt the backup zip file
	r, err := age.NewScryptRecipient(password)
	if err != nil {
		logger.Fatalf("age.NewScryptRecipient failed: %s", err)
	}
	// encrypt keystore content
	zipFile, err := os.Open(zipFilePath)
	if err != nil {
		logger.Fatalf("os.Open(%s) failed: %s", zipFilePath, err)
	}
	defer zipFile.Close()

	encZipPath := fmt.Sprintf("%s.enc", zipFilePath)
	encZipFile, err := os.Create(encZipPath)
	if err != nil {
		logger.Fatalf("os.Create(%s) failed", zipFilePath, err)
	}
	if err := localcrypto.AgeEncrypt([]age.Recipient{r}, zipFile, encZipFile); err != nil {
		logger.Fatalf("AgeEncrypt failed", err)
	}
}

// GetKeystorePassword get password for keystore
func GetKeystorePassword(_password string) (string, error) {
	if _password != "" {
		return _password, nil
	}

	password := os.Getenv("RUM_KSPASSWD")
	if password != "" {
		return password, nil
	}

	return localcrypto.PassphrasePromptForUnlock()
}
