package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/dgraph-io/badger/v3"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/spf13/cobra"
)

var (
	newDataDir string
	param      migrateParam
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "migrate data from badger to boltdb",
	Run: func(cmd *cobra.Command, args []string) {
		if err := migrateAll(); err != nil {
			logger.Fatal(err)
		}
	},
}

type (
	migrateParam struct {
		PeerName   string
		DataDir    string
		NewDataDir string
	}
)

func init() {
	rootCmd.AddCommand(migrateCmd)

	flags := migrateCmd.Flags()
	flags.SortFlags = false

	flags.StringVar(&param.PeerName, "peername", "peer", "peer name")
	flags.StringVar(&param.DataDir, "datadir", "data", "data dir")
	flags.StringVar(&param.NewDataDir, "newdatadir", "newdata", "new data dir")
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

	err = srcDB.View(func(txn *badger.Txn) error {
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

			if len(keys) >= 1000 {
				if err := dstDB.BatchWrite(keys, vals); err != nil {
					return err
				}
				keys = [][]byte{}
				vals = [][]byte{}
			} else {
				keys = append(keys, k)
				vals = append(vals, v)
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
	if err != nil {
		return err
	}
	return nil
}

func migrateAll() error {
	kinds := []string{"db", "appdb", "groups", "pubqueue"}
	for _, kind := range kinds {
		fmt.Printf("migrate %s\n", kind)
		if err := migrateDB(param.PeerName, param.DataDir, kind, param.NewDataDir); err != nil {
			return err
		}
	}

	fmt.Printf("migrate data to %s\n", param.NewDataDir)
	return nil
}
