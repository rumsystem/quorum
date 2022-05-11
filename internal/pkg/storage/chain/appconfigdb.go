package chainstorage

import (
	"errors"
	s "github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func (cs *Storage) UpdateAppConfigTrx(trx *quorumpb.Trx, Prefix ...string) (err error) {
	return cs.UpdateAppConfig(trx.Data, Prefix...)
}

func (cs *Storage) UpdateAppConfig(data []byte, Prefix ...string) (err error) {
	nodeprefix := utils.GetPrefix(Prefix...)
	item := &quorumpb.AppConfigItem{}
	if err := proto.Unmarshal(data, item); err != nil {
		return err
	}
	key := nodeprefix + s.APP_CONFIG_PREFIX + "_" + item.GroupId + "_" + item.Name

	if item.Action == quorumpb.ActionType_ADD {
		chaindb_log.Infof("Add AppConfig item")
		return cs.dbmgr.Db.Set([]byte(key), data)
	} else if item.Action == quorumpb.ActionType_REMOVE {
		chaindb_log.Infof("Remove AppConfig item")
		exist, err := cs.dbmgr.Db.IsExist([]byte(key))
		if !exist {
			if err != nil {
				return err
			}
			return errors.New("AppConfig key not Found")
		}

		return cs.dbmgr.Db.Delete([]byte(key))
	} else {
		return errors.New("Unknown ACTION")
	}
}

// name, type
func (cs *Storage) GetAppConfigKey(groupId string, prefix ...string) ([]string, []string, error) {
	nodeprefix := utils.GetPrefix(prefix...)
	key := nodeprefix + s.APP_CONFIG_PREFIX + "_" + groupId

	var itemName []string
	var itemType []string

	err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		item := quorumpb.AppConfigItem{}
		perr := proto.Unmarshal(v, &item)
		if perr != nil {
			return perr
		}
		itemName = append(itemName, item.Name)
		itemType = append(itemType, item.Type.String())
		return nil
	})
	return itemName, itemType, err
}

func (cs *Storage) GetAppConfigItem(itemKey string, groupId string, Prefix ...string) (*quorumpb.AppConfigItem, error) {
	nodeprefix := utils.GetPrefix(Prefix...)
	key := nodeprefix + s.APP_CONFIG_PREFIX + "_" + groupId + "_" + itemKey

	value, err := cs.dbmgr.Db.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	var config quorumpb.AppConfigItem
	err = proto.Unmarshal(value, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (cs *Storage) GetAllAppConfigInBytes(groupId string, Prefix ...string) ([][]byte, error) {
	nodeprefix := utils.GetPrefix(Prefix...)
	key := nodeprefix + s.APP_CONFIG_PREFIX + "_" + groupId + "_"
	var appConfigByteList [][]byte

	err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		appConfigByteList = append(appConfigByteList, v)
		return nil
	})

	return appConfigByteList, err
}
