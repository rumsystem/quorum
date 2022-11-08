package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/phayes/freeport"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/api"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	"github.com/rumsystem/quorum/testnode"
	"github.com/spf13/cobra"
)

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore rum data from backup file",
	Run: func(cmd *cobra.Command, args []string) {
		if keystorePassword == "" {
			keystorePassword = os.Getenv("RUM_KSPASSWD")
		}
		passwd, err := handlers.GetKeystorePassword(keystorePassword)
		if err != nil {
			logger.Fatalf("handlers.GetKeystorePassword failed: %s", err)
		}

		params := handlers.RestoreParam{
			Peername:    peerName,
			BackupFile:  backupFile,
			Password:    passwd,
			ConfigDir:   configDir,
			KeystoreDir: keystoreDir,
			DataDir:     dataDir,
			SeedDir:     seedDir,
		}
		restore(params, isWasm)
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)

	flags := restoreCmd.Flags()
	flags.SortFlags = false
	flags.StringVar(&peerName, "peername", "peer", "peer name")
	flags.StringVar(&configDir, "configdir", "config", "config directory")
	flags.StringVar(&keystoreDir, "keystoredir", "keystore", "keystore directory")
	flags.StringVar(&dataDir, "datadir", "data", "data directory")
	flags.StringVar(&seedDir, "seeddir", "seeds", "seeds directory")
	flags.StringVar(&keystorePassword, "keystorepass", "", "keystore password")
	flags.StringVar(&backupFile, "file", "", "backup file path")
	flags.BoolVar(&isWasm, "wasm", false, "is wasm")

	restoreCmd.MarkFlagRequired("file")
}

func restore(params handlers.RestoreParam, isWasm bool) {
	var err error
	params.BackupFile, err = filepath.Abs(params.BackupFile)
	if err != nil {
		logger.Fatalf("get absolute path for %s failed: %s", params.BackupFile, err)
	}
	params.ConfigDir, err = filepath.Abs(params.ConfigDir)
	if err != nil {
		logger.Fatalf("get absolute path for %s failed: %s", params.ConfigDir, err)
	}
	params.KeystoreDir, err = filepath.Abs(params.KeystoreDir)
	if err != nil {
		logger.Fatalf("get absolute path for %s failed: %s", params.KeystoreDir, err)
	}
	params.DataDir, _ = filepath.Abs(params.DataDir)
	if err != nil {
		logger.Fatalf("get absolute path for %s failed: %s", params.DataDir, err)
	}
	params.SeedDir, err = filepath.Abs(params.SeedDir)
	if err != nil {
		logger.Fatalf("get absolute path for %s failed: %s", params.SeedDir, err)
	}

	// go to restore directory before restore
	restoreDir := filepath.Dir(params.DataDir)
	if err := utils.EnsureDir(restoreDir); err != nil {
		logger.Fatalf("utils.EnsureDir(%s) failed: %s", restoreDir, err)
	}

	currentDir, err := os.Getwd()
	if err != nil {
		logger.Fatalf("os.Getwd failed: %s", err)
	}

	os.Chdir(restoreDir)
	defer os.Chdir(currentDir)

	if isWasm {
		handlers.RestoreFromWasm(params)
	} else {
		handlers.Restore(params)
	}

	var pidch chan int
	process := os.Args[0]

	apiPort, err := freeport.GetFreePort()
	if err != nil {
		logger.Fatalf("freeport.GetFreePort failed: %s", err)
	}
	testnode.Fork(
		pidch, params.Password, process,
		"fullnode",
		"--peername", params.Peername,
		"--apiport", fmt.Sprintf("%d", apiPort),
		"--configdir", params.ConfigDir,
		"--keystoredir", params.KeystoreDir,
		"--datadir", params.DataDir,
	)
	defer utils.RemoveAll("certs") // NOTE: HARDCODE

	peerBaseUrl := fmt.Sprintf("http://127.0.0.1:%d", apiPort)
	ctx := context.Background()
	checkctx, _ := context.WithTimeout(ctx, 300*time.Second)
	if ok := testnode.CheckApiServerRunning(checkctx, peerBaseUrl); !ok {
		logger.Fatal("api server start failed")
	}

	if utils.DirExist(params.SeedDir) {
		seeds, err := ioutil.ReadDir(params.SeedDir)
		if err != nil {
			logger.Errorf("read seeds directory failed: %s", err)
		}

		for _, seed := range seeds {
			if seed.IsDir() {
				continue
			}

			path := filepath.Join(params.SeedDir, seed.Name())
			seedByte, err := ioutil.ReadFile(path)
			if err != nil {
				logger.Errorf("read seed file failed: %s", err)
				continue
			}

			var seed handlers.CreateGroupResult
			if err := json.Unmarshal(seedByte, &seed); err != nil {
				logger.Errorf("unmarshal seed file failed: %s", err)
				continue
			}

			if _, err := api.JoinGroupByHTTPRequest(peerBaseUrl, &seed); err != nil {
				logger.Errorf("join group %s failed: %s", seed.GroupId, err)
			}
		}
	}

	if _, _, err := testnode.RequestAPI(peerBaseUrl, "/api/quit", "GET", ""); err != nil {
		logger.Fatalf("quit app failed: %s", err)
	}
}
