package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/dgraph-io/badger/v3"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

const (
	txMaxSize = 1024 * 1024 * 50
)

var (
	_migrateParam dbParam
	_compactParam dbParam

	kinds = []string{"db", "appdb", "groups", "pubqueue"} // FIXME: hardcode
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
)

type (
	dbParam struct {
		PeerName   string
		DataDir    string
		NewDataDir string
	}
)

func init() {
	dbCmd.AddCommand(migrateCmd)
	dbCmd.AddCommand(compactCmd)
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

			keys = append(keys, k)
			vals = append(vals, v)

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

	for _, kind := range kinds {
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

	for _, kind := range kinds {
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
