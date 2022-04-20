package stats

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"
)

const (
	dbPathSuffix = "_stats"
)

type StatsDB struct {
	db        storage.QuorumStorage
	localPeer peer.ID
}

var statsDB *StatsDB

func GetStatsDB() *StatsDB {
	return statsDB
}

func InitDB(path string, localPeer peer.ID) error {
	if statsDB == nil {
		db, err := openStatsDB(path)
		if err != nil {
			return err
		}
		statsDB = &StatsDB{db: db, localPeer: localPeer}
	}

	return nil
}

func GetLocalPeerID() peer.ID {
	if statsDB != nil {
		return statsDB.localPeer
	}

	panic("you must invoke stats.InitDB first")
}

func (sdb *StatsDB) AddNetworkLog(log *NetworkStats) error {
	if log.CreatedAt == nil {
		now := time.Now()
		log.CreatedAt = &now
	}

	key := log.GetDBKey()
	value, err := json.Marshal(log)
	if err != nil {
		return err
	}

	if sdb == nil || sdb.db == nil {
		return errors.New("sdb or sdb.db is nil")
	}

	return sdb.db.Set([]byte(key), value)
}

func (sdb *StatsDB) ParseNetworkLog(start, end *time.Time) (*NetworkStatsSummary, error) {
	var prefix string

	if end == nil {
		now := time.Now()
		end = &now
	}

	if start != nil && end != nil {
		startStr := start.Format(layout)
		endStr := end.Format(layout)
		common := utils.LongestCommonPrefix([]string{startStr, endStr})
		prefix = GetDBKeyPrefixByStr(common)
	} else if start == nil {
		prefix = "" // iterate over all keys
	}

	result := NewNetworkStatsSummary()
	err := sdb.db.PrefixForeach([]byte(prefix), func(k []byte, v []byte, err error) error {
		if err != nil {
			logger.Errorf("sdb.db.PrefixForeach failed: %s", err)
			return err
		}

		// make sure the datetime of this item is between start and end
		key, err := ParseDBKey(string(k))
		if err != nil {
			logger.Errorf("ParseDBKey failed: %s", err)
			return err
		}

		if key.Datetime.After(*end) || (start != nil && key.Datetime.Before(*start)) {
			return nil // skip
		}

		// parse value
		var stats NetworkStats
		if err := json.Unmarshal(v, &stats); err != nil {
			logger.Errorf("json.Unmarshal(%s) failed: %s", v, err)
			return err
		}
		summary := stats.ToNetworkStatsSummaryItem()
		result.Add(*summary)

		return nil
	})

	return result, err
}
