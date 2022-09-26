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
	"strings"

	"filippo.io/age"
	"github.com/rumsystem/quorum/internal/pkg/appdata"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"

	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/rumsystem/keystore/pkg/crypto"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
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
	return filepath.Join(dstPath, "wasm/backup.json")
}

func getBlockPrefixKey() string {
	return nodename + "_" + storage.BLK_PREFIX + "_"
}

// BackupForWasm will backup keystore in wasm known format
func BackupForWasm(param BackupParam) {
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

	/*
			   wasm need a single file in json format

			   ```
		     {"keystore": [], "seeds": []}
			   ```
	*/
	wasmDstPath := getWasmBackupPath(dstPath)
	wasmKeystoreContent := []string{}
	if err := filepath.Walk(param.KeystoreDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
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
		pair["value"] = base64.StdEncoding.EncodeToString(keyBytes)
		if err != nil {
			return err
		}
		if strings.HasPrefix(key, crypto.Sign.Prefix()) {
			key, err := ethkeystore.DecryptKey(keyBytes, password)
			if err != nil {
				return err
			}
			privKey := key.PrivateKey
			addr := ethcrypto.PubkeyToAddress(privKey.PublicKey)
			// Make sure we're really operating on the requested key (no swap attacks)
			if key.Address != addr {
				return fmt.Errorf("key content mismatch: have account %x, want %x", key.Address, addr)
			}
			pair["addr"] = addr.String()
		}
		kvBytes, err := json.Marshal(pair)

		output := new(bytes.Buffer)
		if err := crypto.AgeEncrypt([]age.Recipient{r}, bytes.NewReader(kvBytes), output); err != nil {
			return err
		}
		encryptedKvBytes, err := ioutil.ReadAll(output)
		if err != nil {
			return err
		}
		res := base64.StdEncoding.EncodeToString(encryptedKvBytes)
		wasmKeystoreContent = append(wasmKeystoreContent, res)
		return nil
	}); err != nil {
		logger.Fatalf("export keystore to wasm failed: %s", err)
	}

	backupObj := QuorumWasmExportObject{}
	backupObj.Keystore = wasmKeystoreContent

	// ExportAllGroupSeeds
	dataPath := GetDataPath(param.DataDir, param.Peername)
	appdb, err := appdata.CreateAppDb(dataPath)
	if err != nil {
		logger.Fatalf("appdata.CreateAppDb failed: %s", err)
	}
	seeds, err := GetAllGroupSeeds(appdb)
	backupObj.Seeds = seeds

	if err := os.MkdirAll(filepath.Dir(wasmDstPath), 0770); err != nil {
		logger.Fatalf("create wasm keystore path failed: %s", err)
	}

	f, err := os.Create(wasmDstPath)
	if err != nil {
		logger.Fatalf("create wasm keystore file failed: %s", err)
	}
	defer f.Close()

	backupBytes, err := json.Marshal(backupObj)

	f.Write(backupBytes)

	logger.Infof("success! backup file: %s", wasmDstPath)
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
	seedDstPath := getSeedBackupPath(dstPath)
	SaveAllGroupSeeds(appdb, seedDstPath)

	// backup block
	blockDstPath := getBlockBackupPath(dstPath)
	BackupBlock(param.DataDir, param.Peername, blockDstPath)

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
	if err := loadAndDecryptTrx(blockDstPath, seedDstPath, ks); err != nil {
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
