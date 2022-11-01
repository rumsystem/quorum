package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/adrg/xdg"
	"github.com/rumsystem/quorum/cmd/cli/config"
	"github.com/rumsystem/quorum/internal/pkg/storage"
)

type QuorumDataCache struct {
	db *storage.Store
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
		ctx := context.Background()
		bucket := "cache"
		db, err := storage.NewStore(ctx, path, bucket)
		if err != nil {
			config.Logger.Errorf("Failed to open cache db: %s", err.Error())
		}
		QCache = &QuorumDataCache{db}
		config.Logger.Infof("cache db opened")
	}
}

func (cache *QuorumDataCache) Set(key []byte, value []byte) {
	if cache != nil && cache.db != nil {
		if err := cache.db.Set(key, value); err != nil {
			config.Logger.Errorf("Failed to Set: %s\n", err.Error())
		} else {
			config.Logger.Infof("cache %s setted", string(key))
		}
	}
}

func (cache *QuorumDataCache) Get(key []byte) ([]byte, error) {
	if cache != nil && cache.db != nil {
		return cache.db.Get(key)
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
