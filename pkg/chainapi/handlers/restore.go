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
	"github.com/rumsystem/quorum/pkg/crypto"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"
)

type RestoreParam struct {
	Peername    string `json:"peername" validate:"required"`
	Password    string `json:"password" validate:"required"`
	BackupFile  string `json:"backup_file" validate:"required"`
	KeystoreDir string `json:"keystore_dir" validate:"required"`
	ConfigDir   string `json:"config_dir" validate:"required"`
	SeedDir     string `json:"seed_dir" validate:"required"`
	DataDir     string `json:"data_dir" validate:"required"`
}

// Restore restores the keystore and config from backup data
func Restore(params RestoreParam) {
	encZipPath := params.BackupFile

	// check restore path
	if exist := utils.FileExist(encZipPath); !exist {
		logger.Fatalf("can not find %s", encZipPath)
	}

	// age identities
	identities := []age.Identity{
		&localcrypto.LazyScryptIdentity{Password: params.Password},
	}

	encZipFile, err := os.Open(encZipPath)
	if err != nil {
		logger.Fatalf("os.Open(%s) failed: %s", encZipPath, err)
	}
	defer encZipFile.Close()

	zipFile, err := age.Decrypt(encZipFile, identities...)
	if err != nil {
		logger.Fatalf("decrypt encrypted zip file failed: %v", err)
	}
	zipFilePath := strings.Replace(encZipPath, ".enc", "", 1)
	absZipFilePath, err := filepath.Abs(zipFilePath)
	if err != nil {
		logger.Fatalf("filepath.Abs(%s) failed: %s", zipFilePath, err)
	}
	defer utils.RemoveAll(absZipFilePath)

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(zipFile)
	if err != nil {
		logger.Fatalf("buf.ReadFrom failed: %s", err)
	}
	if err := ioutil.WriteFile(absZipFilePath, buf.Bytes(), 0600); err != nil {
		logger.Fatalf("ioutil.WriteFile failed: %s", err)
	}

	absUnZipDir := utils.PathTrimExt(absZipFilePath)
	defer utils.RemoveAll(absUnZipDir)
	if err := utils.Unzip(zipFilePath, absUnZipDir); err != nil {
		logger.Fatalf("unzip backup zip archive failed: %v", err)
	}

	// copy config dir
	if err := utils.CheckAndCreateDir(params.ConfigDir); err != nil {
		logger.Fatalf("create directory %s failed: %s", params.ConfigDir, err)
	}
	srcConfigDir := getConfigBackupPath(absUnZipDir)
	if err := utils.Copy(srcConfigDir, params.ConfigDir); err != nil {
		logger.Fatalf("copy %s => %s failed: %s", srcConfigDir, params.ConfigDir, err)
	}

	// copy keystore dir
	if err := utils.CheckAndCreateDir(params.KeystoreDir); err != nil {
		logger.Fatalf("create directory %s failed: %s", params.KeystoreDir, err)
	}
	srcKeystoreDir := getKeystoreBackupPath(absUnZipDir)
	if err := utils.Copy(srcKeystoreDir, params.KeystoreDir); err != nil {
		logger.Fatalf("copy %s => %s failed: %s", srcKeystoreDir, params.KeystoreDir, err)
	}

	// copy seed dir
	if err := utils.CheckAndCreateDir(params.SeedDir); err != nil {
		logger.Fatalf("create directory %s failed: %s", params.SeedDir, err)
	}
	srcSeedDir := getSeedBackupPath(absUnZipDir)
	if err := utils.Copy(srcSeedDir, params.SeedDir); err != nil {
		logger.Fatalf("copy %s => %s failed: %s", srcSeedDir, params.SeedDir, err)
	}

	// restore block db
	srcBlockDBDir := getBlockBackupPath(absUnZipDir)
	dstBlockDBDir := getBlockRestorePath(params.Peername, params.DataDir)
	if err := restoreBlockDB(srcBlockDBDir, dstBlockDBDir); err != nil {
		logger.Fatalf("restoreBlockDB(%s) failed: %s", srcBlockDBDir, err)
	}
}

// from wasm export file(keystore)
func RestoreFromWasm(param RestoreParam) {
	wasmDstPath := param.BackupFile

	readFile, err := os.Open(wasmDstPath)
	defer readFile.Close()
	if err != nil {
		logger.Fatalf("failed to restore from wasm backup file: %s", err)
	}

	identities := []age.Identity{
		&crypto.LazyScryptIdentity{param.Password},
	}

	backupObj := QuorumWasmExportObject{}
	backupBytes, err := ioutil.ReadAll(readFile)
	if err != nil {
		logger.Fatalf("failed to restore from wasm backup file: %s", err)
	}
	err = json.Unmarshal(backupBytes, &backupObj)
	if err != nil {
		logger.Fatalf("failed to decode backup file: %s", err)
	}

	nodeoptions, err := options.InitNodeOptions(param.ConfigDir, param.Peername)
	if err != nil {
		logger.Fatalf(err.Error())
	}

	// restore keystore files
	for _, ks := range backupObj.Keystore {
		enc, err := base64.StdEncoding.DecodeString(ks)
		if err != nil {
			logger.Fatalf("base64 decode config data failed: %s", err)
		}

		r, err := age.Decrypt(bytes.NewReader(enc), identities...)
		if err != nil {
			logger.Fatalf("decrypt config data failed: %v", err)
		}

		kvBytes, err := ioutil.ReadAll(r)
		if err != nil {
			logger.Fatalf("ioutil.ReadAll config failed: %v", err)
		}
		pair := make(map[string]interface{})
		err = json.Unmarshal(kvBytes, &pair)
		if err != nil {
			logger.Fatalf("failed to restore from wasm keystore: %s", err)
		}
		k := pair["key"].(string)
		v, _ := base64.StdEncoding.DecodeString(pair["value"].(string))
		logger.Infof("Loading %s", k)

		if strings.HasPrefix(k, crypto.Sign.Prefix()) {
			addr := pair["addr"].(string)
			keyName := strings.ReplaceAll(k, crypto.Sign.Prefix(), "")
			nodeoptions.SetSignKeyMap(keyName, addr)
		}

		ksPath := filepath.Join(param.KeystoreDir, k)
		if err := os.MkdirAll(filepath.Dir(ksPath), 0770); err != nil {
			logger.Fatalf("create wasm keystore path failed: %s", err)
		}
		f, err := os.Create(ksPath)
		if err != nil {
			logger.Fatalf("create wasm keystore file failed: %s", err)
		}
		f.Write(v)
		f.Close()
		logger.Infof("OK")
	}

	// restore seeds
	if err := utils.CheckAndCreateDir(param.SeedDir); err != nil {
		logger.Fatalf("create directory %s failed: %s", param.SeedDir, err)
	}

	for _, seed := range backupObj.Seeds {
		seedByte, err := json.MarshalIndent(seed, "", "  ")
		if err != nil {
			logger.Fatalf("marshal group seed failed: %s", err)
		}

		path := filepath.Join(param.SeedDir, fmt.Sprintf("%s.json", seed.GroupId))
		if err := ioutil.WriteFile(path, seedByte, 0644); err != nil {
			logger.Fatalf("write group seed failed: %s", err)
		}
	}
}

// restoreBlockDB restore block data to `data/{peerName}_db`
func restoreBlockDB(srcBlockDBDir string, dstDir string) error {
	if err := utils.EnsureDir(dstDir); err != nil {
		return fmt.Errorf("utils.EnsureDir(%s) failed: %s", dstDir, err)
	}

	srcDB := storage.QSBadger{}
	err := srcDB.Init(srcBlockDBDir)
	if err != nil {
		return fmt.Errorf("srcDB.Init failed: %s", err)
	}
	defer srcDB.Close()

	dstDB := storage.QSBadger{}
	err = dstDB.Init(dstDir)
	if err != nil {
		return fmt.Errorf("dstDB.Init failed: %s", err)
	}
	defer dstDB.Close()

	key := getBlockPrefixKey()
	err = srcDB.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}

		if err := dstDB.Set(k, v); err != nil {
			return fmt.Errorf("restoreDB.Set(%s) failed: %s", key, err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("backupDB.PrefixForeach failed: %s", err)
	}

	return nil
}
