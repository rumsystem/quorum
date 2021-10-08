package cache

import (
	"errors"
	"fmt"

	"github.com/adrg/xdg"
	"github.com/rumsystem/quorum/cmd/cli/config"

	badger "github.com/dgraph-io/badger/v3"
	badgerOptions "github.com/dgraph-io/badger/v3/options"
)

type QuorumDataCache struct {
	db *badger.DB
}

var QCache *QuorumDataCache

func GetUserProfileKey(group string, pubkey string) string {
	return fmt.Sprintf("profile_%s_%s", group, pubkey)
}

func Init() {
	// $XDG_DATA_HOME/rumcli/data
	path, err := xdg.DataFile("rumcli/data")
	if err != nil {
		config.Logger.Fatalf(err.Error())
	}
	if QCache == nil {
		db, err := badger.Open(badger.DefaultOptions(path).WithCompression(badgerOptions.Snappy))
		if err != nil {
			config.Logger.Errorf("Failed to open db: %s", err.Error())
		}
		QCache = &QuorumDataCache{db}
		config.Logger.Infof("cache db opened")
	}
}

func (cache *QuorumDataCache) Set(key []byte, value []byte) {
	if cache != nil && cache.db != nil {
		err := cache.db.Update(func(txn *badger.Txn) error {
			e := badger.NewEntry(key, value)
			err := txn.SetEntry(e)
			return err
		})

		if err != nil {
			config.Logger.Errorf("Failed to Set: %s\n", err.Error())
		} else {
			config.Logger.Infof("cache %s setted", string(key))
		}
	}
}

func (cache *QuorumDataCache) Get(key []byte) ([]byte, error) {
	if cache != nil && cache.db != nil {
		var ret []byte
		err := cache.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get(key)
			if err != nil {
				return err
			}

			ret, err = item.ValueCopy(nil)
			if err != nil {
				return err
			}
			return err
		})
		return ret, err
	}
	return nil, errors.New("Not found")
}

func (cache *QuorumDataCache) StartSync(interval int) {
	if cache != nil {
	}
}

func (cache *QuorumDataCache) StopSync() {
	if cache != nil {
	}
}

func Shutdown() {
	if QCache != nil && QCache.db != nil {
		QCache.db.Close()
	}
}
