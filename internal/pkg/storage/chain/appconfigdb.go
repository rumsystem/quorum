package chainstorage

import (
	"errors"

	s "github.com/rumsystem/quorum/internal/pkg/storage"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func (cs *Storage) UpdateAppConfig(data []byte, prefix ...string) (err error) {
	item := &quorumpb.AppConfigItem{}
	if err := proto.Unmarshal(data, item); err != nil {
		return err
	}
	key := s.GetAppConfigKey(item.GroupId, item.Name, prefix...)

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
	key := s.GetAppConfigPrefix(groupId, prefix...)

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

func (cs *Storage) GetAppConfigItem(itemKey string, groupId string, prefix ...string) (*quorumpb.AppConfigItem, error) {
	key := s.GetAppConfigKey(groupId, itemKey, prefix...)
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

func (cs *Storage) GetAllAppConfigInBytes(groupId string, prefix ...string) ([][]byte, error) {
	var appConfigByteList [][]byte
	key := s.GetAppConfigPrefix(groupId, prefix...)

	err := cs.dbmgr.Db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		appConfigByteList = append(appConfigByteList, v)
		return nil
	})

	return appConfigByteList, err
}
