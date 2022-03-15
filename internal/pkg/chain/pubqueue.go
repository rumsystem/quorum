package chain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/conn"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"github.com/rumsystem/quorum/internal/pkg/storage"
)

type PublishQueueItem struct {
	// also stored in the key for quick indexing
	groupId string
	state   string

	// in value only
	retryCount int
	updateAt   int64
	trx        *quorumpb.Trx
}

const (
	PublishQueueItemStatePending = "PENDING"
	PublishQueueItemStateSuccess = "SUCCESS"
	PublishQueueItemStateFail    = "FAIL"
	PUBQUEUE_PREFIX              = "PUBQUEUE"
	MAX_RETRY_COUNT              = 10
)

func (item *PublishQueueItem) GetKey() ([]byte, error) {
	// PREFIX_STATE_GROUPID_TRXID

	validStates := []string{PublishQueueItemStatePending, PublishQueueItemStateSuccess, PublishQueueItemStateFail}

	if item.groupId == "" {
		return nil, fmt.Errorf("group id can not be empty")
	}

	stateValid := false
	for _, s := range validStates {
		if item.state == s {
			stateValid = true
		}
	}
	if !stateValid {
		return nil, fmt.Errorf("state(%s) is invalid", item.state)
	}

	if item.trx == nil {
		return nil, fmt.Errorf("trx can not be nil")
	}

	key := fmt.Sprintf("%s_%s_%s_%s", PUBQUEUE_PREFIX, item.state, item.groupId, item.trx.TrxId)
	return []byte(key), nil
}

func (item *PublishQueueItem) GetValue() ([]byte, error) {
	return json.Marshal(item)
}

func ParsePublishQueueItem(k, v []byte) (*PublishQueueItem, error) {
	key := string(k)
	keys := strings.Split(key, "_")
	if len(keys) != 4 || keys[0] != PUBQUEUE_PREFIX {
		return nil, fmt.Errorf("invalid key(%s)", key)
	}
	item := PublishQueueItem{}
	err := json.Unmarshal(v, &item)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

type PublishQueueWatcher struct {
	db storage.QuorumStorage
}

var publishQueueWatcher PublishQueueWatcher = PublishQueueWatcher{}

func InitPublishQueueWatcher(done chan bool, db storage.QuorumStorage) {

	publishQueueWatcher.db = db

	// hard coded to 10s
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				doRefresh()
			}
		}
	}()
}

func doRefresh() {
	if publishQueueWatcher.db == nil {
		return
	}

	// remove succeed trx (wasm compatable)
	publishQueueWatcher.db.PrefixDelete([]byte(fmt.Sprintf("%s_%s_", PUBQUEUE_PREFIX, PublishQueueItemStateSuccess)))

	publishQueueWatcher.db.PrefixForeach([]byte(PUBQUEUE_PREFIX), func(k []byte, v []byte, err error) error {
		if err != nil {
			chain_log.Warnf("<pubqueue>: %s", err.Error())
			// continue
			return nil
		}
		item, err := ParsePublishQueueItem(k, v)
		if err != nil {
			chain_log.Warnf("<pubqueue>: %s", err.Error())
			return nil
		}

		switch item.state {
		case PublishQueueItemStatePending:
			// check trx state, update, so it will be removed or retied in next poll
			groupmgr := GetGroupMgr()
			if group, ok := groupmgr.Groups[item.groupId]; ok {
				trx, _, err := group.GetTrx(item.trx.TrxId)
				if err != nil {
					chain_log.Errorf("<pubqueue>: %s", err.Error())
				} else {
					chain_log.Infof("<pubqueue>: got trx %v", trx)
					item.state = PublishQueueItemStateSuccess
				}

			}
		case PublishQueueItemStateFail:
			// retry then mark as pending
			// TODO: error handling
			groupmgr := GetGroupMgr()
			if group, ok := groupmgr.Groups[item.groupId]; ok {
				muser := group.ChainCtx.Consensus.User().(*MolassesUser)
				muser.sendTrx(item.trx, conn.ProducerChannel)
				item.state = PublishQueueItemStatePending
			}

		default:
		}

		newK, err := item.GetKey()
		if err != nil {
			chain_log.Errorf("<pubqueue>: %s", err.Error())
			return nil
		}
		newV, err := item.GetValue()
		if err != nil {
			chain_log.Errorf("<pubqueue>: %s", err.Error())
			return nil
		}
		item.updateAt = time.Now().Unix()
		publishQueueWatcher.db.Set(newK, newV)
		return nil
	})

}

func TrxEnqueue(trx *quorumpb.Trx) {
	// TODO: store/update the trx in db after published
}
