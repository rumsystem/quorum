//go:build !js
// +build !js

package handlers

import (
	"fmt"
	"os"
	"path/filepath"

	"filippo.io/age"
	"github.com/rumsystem/quorum/internal/pkg/appdata"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"

	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
)

type BackupParam struct {
	Peername     string `json:"peername" validate:"required"`
	Password     string `json:"password" validate:"required"`
	BackupFile   string `json:"backup_file" validate:"required"`
	KeystoreDir  string `json:"keystore_dir" validate:"required"`
	KeystoreName string `json:"keystore_name" validate:"required"`
	ConfigDir    string `json:"config_dir" validate:"required"`
	SeedDir      string `json:"seed_dir" validate:"required"`
	DataDir      string `json:"data_dir" validate:"required"`
}

func GetDataPath(dataDir, peerName string) string {
	return filepath.Join(dataDir, peerName)
}

func getSeedBackupPath(dstPath string) string {
	return filepath.Join(dstPath, "seeds")
}

func getDataBackupPath(dstPath string, peerName string) string {
	return filepath.Join(dstPath, "data", peerName)
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

func getBlockPrefixKey() string {
	return nodename + "_" + storage.BLK_PREFIX + "_"
}

// Backup backup block from data db and {config,keystore,seeds} directory
func Backup(param BackupParam) {
	// get keystore password
	password, err := GetKeystorePassword(param.Password)
	if err != nil {
		logger.Fatalf("handlers.GetKeystorePassword failed: %s", err)
	}

	// check keystore signature and encrypt
	if err := CheckSignAndEncryptWithKeystore(param.KeystoreName, param.KeystoreDir, param.ConfigDir, param.Peername, password); err != nil {
		logger.Fatalf("check keystore failed: %s", err)
	}

	dstPath := param.BackupFile
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
	if err := utils.Copy(param.ConfigDir, configDstPath); err != nil {
		logger.Fatalf("copy %s => %s failed: %s", param.ConfigDir, dstPath, err)
	}

	// backup keystore
	keystoreDstPath := getKeystoreBackupPath(dstPath)
	if err := utils.Copy(param.KeystoreDir, keystoreDstPath); err != nil {
		logger.Fatalf("copy %s => %s failed: %s", param.KeystoreDir, dstPath, err)
	}

	// SaveAllGroupSeeds
	dataPath := GetDataPath(param.DataDir, param.Peername)
	appdb, err := appdata.CreateAppDb(dataPath)
	if err != nil {
		logger.Fatalf("appdata.CreateAppDb failed: %s", err)
	}
	defer appdb.Db.Close()

	seedDstPath := getSeedBackupPath(dstPath)
	SaveAllGroupSeeds(appdb, seedDstPath)

	// backup block
	dataDstPath := getDataBackupPath(dstPath, param.Peername)
	BackupBlock(param.DataDir, param.Peername, dataDstPath)

	// zip backup directory
	zipFilePath := fmt.Sprintf("%s.zip", dstPath)
	defer utils.RemoveAll(dstPath)
	defer utils.RemoveAll(zipFilePath)
	if err := utils.ZipDir(dstPath, zipFilePath); err != nil {
		logger.Fatalf("utils.ZipDir(%s, %s) failed: %s", dstPath, zipFilePath, err)
	}

	// check keystore signature and encrypt
	if err := CheckSignAndEncryptWithKeystore(param.KeystoreName, keystoreDstPath, configDstPath, param.Peername, password); err != nil {
		logger.Fatalf("check keystore failed: %s", err)
	}

	// load keystore and try to decrypt trx data
	nodeoptions, err := options.InitNodeOptions(configDstPath, param.Peername)
	if err != nil {
		logger.Fatalf("load restored config failed: %s", err)
	}
	ks, _, err := localcrypto.InitDirKeyStore(param.KeystoreName, keystoreDstPath)
	if err != nil {
		logger.Fatalf("init restored keystore failed: %s", err)
	}
	ks.Unlock(nodeoptions.SignKeyMap, password)
	if err := loadAndDecryptTrx(dataDstPath, seedDstPath, ks); err != nil {
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

	logger.Infof("success! backup file: %s", encZipPath)
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
