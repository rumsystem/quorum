package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/dgraph-io/badger/v3"
	"github.com/rumsystem/quorum/internal/pkg/appdata"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

const (
	txMaxSize = 1024 * 1024 * 50
)

var (
	_migrateParam    dbParam
	_compactParam    dbParam
	_saveSeedParam   saveSeedParam
	_resetAppdbParam resetAppdbParam

	_migrateDbKinds = []string{"db", "appdb", "groups"}             // FIXME: hardcode
	_compactDbKinds = []string{"db", "appdb", "groups", "pubqueue"} // FIXME: hardcode
)

var (
	dbCmd = &cobra.Command{
		Use:              "db",
		Short:            "database tool, migrate or compact",
		TraverseChildren: true,
	}

	migrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "migrate data from badger to boltdb",
		Run: func(cmd *cobra.Command, args []string) {
			if err := migrateAll(); err != nil {
				logger.Fatal(err)
			}
		},
	}

	compactCmd = &cobra.Command{
		Use:   "compact",
		Short: "compact data",
		Run: func(cmd *cobra.Command, args []string) {
			if err := compactAll(); err != nil {
				logger.Fatal(err)
			}
		},
	}

	seedCmd = &cobra.Command{
		Use:   "seed",
		Short: "seed tool",
	}

	appdbCmd = &cobra.Command{
		Use:   "appdb",
		Short: "appdb tool",
	}

	saveSeedCmd = &cobra.Command{
		Use:   "save",
		Short: "save seed to appdb",
		Run: func(cmd *cobra.Command, args []string) {
			if err := saveSeed(&_saveSeedParam); err != nil {
				logger.Fatal(err)
			}
		},
	}

	resetAppdbCmd = &cobra.Command{
		Use:   "reset",
		Short: "reset appdb",
		Run: func(cmd *cobra.Command, args []string) {
			if err := resetAppdb(&_resetAppdbParam); err != nil {
				logger.Fatal(err)
			}
		},
	}
)

type (
	dbParam struct {
		PeerName   string
		DataDir    string
		NewDataDir string
	}

	saveSeedParam struct {
		PeerName string
		DataDir  string
		SeedPath string // seed json file path
		SeedURL  string // seed url
	}

	resetAppdbParam struct {
		PeerName string
		DataDir  string
	}
)

func init() {
	dbCmd.AddCommand(migrateCmd)
	dbCmd.AddCommand(compactCmd)
	dbCmd.AddCommand(seedCmd)
	dbCmd.AddCommand(appdbCmd)

	seedCmd.AddCommand(saveSeedCmd)
	appdbCmd.AddCommand(resetAppdbCmd)

	rootCmd.AddCommand(dbCmd)

	// migrate
	migrateFlags := migrateCmd.Flags()
	migrateFlags.SortFlags = false

	migrateFlags.StringVar(&_migrateParam.PeerName, "peername", "peer", "peer name")
	migrateFlags.StringVar(&_migrateParam.DataDir, "datadir", "data", "data dir")
	migrateFlags.StringVar(&_migrateParam.NewDataDir, "newdatadir", "", "new data dir")
	migrateCmd.MarkFlagRequired("newdatadir")

	// compact
	compactFlags := compactCmd.Flags()
	compactFlags.SortFlags = false

	compactFlags.StringVar(&_compactParam.PeerName, "peername", "peer", "peer name")
	compactFlags.StringVar(&_compactParam.DataDir, "datadir", "data", "data dir")
	compactFlags.StringVar(&_compactParam.NewDataDir, "newdatadir", "", "new data dir")
	migrateCmd.MarkFlagRequired("newdatadir")

	// save seed
	saveSeedFlags := saveSeedCmd.Flags()
	saveSeedFlags.SortFlags = false
	saveSeedFlags.StringVar(&_saveSeedParam.PeerName, "peername", "peer", "peer name")
	saveSeedFlags.StringVar(&_saveSeedParam.DataDir, "datadir", "data", "data dir")
	saveSeedFlags.StringVar(&_saveSeedParam.SeedPath, "seedpath", "", "seed json file")
	saveSeedFlags.StringVar(&_saveSeedParam.SeedURL, "seedurl", "", "seed url")

	// reset appdb
	resetAppdbFlags := resetAppdbCmd.Flags()
	resetAppdbFlags.SortFlags = false
	resetAppdbFlags.StringVar(&_resetAppdbParam.PeerName, "peername", "peer", "peer name")
	resetAppdbFlags.StringVar(&_resetAppdbParam.DataDir, "datadir", "data", "data dir")
}

func openBadgerDB(dbDir string) (*badger.DB, error) {
	db, err := badger.Open(badger.DefaultOptions(dbDir))
	if err != nil {
		return nil, err
	}
	return db, nil
}

func migrateDB(peerName, dataDir, kind, newDataDir string) error {
	srcPath := filepath.Join(dataDir, fmt.Sprintf("%s_%s", peerName, kind))
	srcDB, err := openBadgerDB(srcPath)
	if err != nil {
		return err
	}
	defer srcDB.Close()

	dstPath := filepath.Join(newDataDir, peerName)
	dstDB, err := storage.NewStore(context.Background(), dstPath, kind)
	if err != nil {
		return err
	}
	defer dstDB.Close()

	return srcDB.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 1000
		it := txn.NewIterator(opts)
		defer it.Close()

		keys := [][]byte{}
		vals := [][]byte{}
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.KeyCopy(nil)
			v, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}

			if kind == "appdb" {
				// for appdb, just migrate group seed
				if appdata.IsGroupSeedKey(k) {
					keys = append(keys, k)
					vals = append(vals, v)
				}
			} else {
				keys = append(keys, k)
				vals = append(vals, v)
			}

			if len(keys) >= 1000 {
				if err := dstDB.BatchWrite(keys, vals); err != nil {
					return err
				}
				keys = [][]byte{}
				vals = [][]byte{}
			}
		}

		if len(keys) > 0 {
			if err := dstDB.BatchWrite(keys, vals); err != nil {
				return err
			}
			keys = [][]byte{}
			vals = [][]byte{}
		}

		return nil
	})
}

func migrateAll() error {
	_dbParam := _migrateParam

	for _, kind := range _migrateDbKinds {
		fmt.Printf("migrate %s\n", kind)
		if err := migrateDB(_dbParam.PeerName, _dbParam.DataDir, kind, _dbParam.NewDataDir); err != nil {
			return err
		}
	}

	fmt.Printf("migrate data to %s\n", _dbParam.NewDataDir)
	return nil
}

func compactAll() error {
	_dbParam := _compactParam
	srcBasePath := filepath.Join(_dbParam.DataDir, peerName)
	dstBasePath := filepath.Join(_dbParam.NewDataDir, peerName)

	for _, kind := range _compactDbKinds {
		fmt.Printf("compact %s\n", kind)
		srcDB, err := storage.OpenDB(srcBasePath, kind)
		if err != nil {
			return err
		}

		dstDB, err := storage.OpenDB(dstBasePath, kind)
		if err != nil {
			return err
		}

		if err := bbolt.Compact(dstDB, srcDB, txMaxSize); err != nil {
			return err
		}
	}

	fmt.Printf("please use new data directory: %s\n", _dbParam.NewDataDir)

	return nil
}

func saveSeed(param *saveSeedParam) error {
	if param.SeedPath == "" && param.SeedURL == "" {
		return errors.New("you must specify command line option `--seedpath` or `--seedurl`")
	}

	seed := &handlers.GroupSeed{}

	if param.SeedPath != "" {
		// parse seed
		if !utils.FileExist(param.SeedPath) {
			return fmt.Errorf("can not find seed file: %s", param.SeedPath)
		}

		seedContent, err := ioutil.ReadFile(param.SeedPath)
		if err != nil {
			return fmt.Errorf("read seed from file failed: %s", err)
		}

		if err := json.Unmarshal(seedContent, seed); err != nil {
			return fmt.Errorf("invalid group seed: %s", err)
		}
	} else {
		var err error
		seed, _, err = handlers.UrlToGroupSeed(param.SeedURL)
		if err != nil {
			return fmt.Errorf("invalid seed url: %s", err)
		}
	}

	path := filepath.Join(param.DataDir, param.PeerName)
	appdb, err := appdata.CreateAppDb(path)
	if err != nil {
		return fmt.Errorf("open appdb failed: %s", err)
	}

	pbGroupSeed := handlers.ToPbGroupSeed(*seed)
	if err := appdb.SetGroupSeed(&pbGroupSeed); err != nil {
		return fmt.Errorf("save group seed failed: %s", err)
	}

	return nil
}

func resetAppdb(param *resetAppdbParam) error {
	path := filepath.Join(param.DataDir, param.PeerName)
	appdb, err := appdata.CreateAppDb(path)
	if err != nil {
		return fmt.Errorf("open appdb failed: %s", err)
	}

	if err := appdb.Reset(); err != nil {
		return fmt.Errorf("reset appdb failed: %s", err)
	}

	return nil
}
