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
	GroupId string

	// in value only
	State      string
	RetryCount int
	UpdateAt   int64 `json:"UpdateAt,string"`
	Trx        *quorumpb.Trx
}

const (
	PublishQueueItemStatePending = "PENDING"
	PublishQueueItemStateSuccess = "SUCCESS"
	PublishQueueItemStateFail    = "FAIL"
	PUBQUEUE_PREFIX              = "PUBQUEUE"
	MAX_RETRY_COUNT              = 10
)

var autoAck bool = false

func SetAutoAck(ack bool) {
	autoAck = ack
}

func (item *PublishQueueItem) GetKey() ([]byte, error) {
	// PREFIX_GROUPID_TRXID
	validStates := []string{PublishQueueItemStatePending, PublishQueueItemStateSuccess, PublishQueueItemStateFail}

	if item.GroupId == "" {
		return nil, fmt.Errorf("group id can not be empty")
	}

	stateValid := false
	for _, s := range validStates {
		if item.State == s {
			stateValid = true
		}
	}
	if !stateValid {
		return nil, fmt.Errorf("state(%s) is invalid", item.State)
	}

	if item.Trx == nil {
		return nil, fmt.Errorf("trx can not be nil")
	}

	key := fmt.Sprintf("%s_%s_%s", PUBQUEUE_PREFIX, item.GroupId, item.Trx.TrxId)
	return []byte(key), nil
}

func (item *PublishQueueItem) GetValue() ([]byte, error) {
	return json.Marshal(item)
}

func ParsePublishQueueItem(k, v []byte) (*PublishQueueItem, error) {
	key := string(k)
	keys := strings.Split(key, "_")
	if len(keys) != 3 || keys[0] != PUBQUEUE_PREFIX {
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

	running bool
}

func (watcher *PublishQueueWatcher) UpsertItem(item *PublishQueueItem) error {
	newK, err := item.GetKey()
	if err != nil {
		return err
	}
	item.UpdateAt = time.Now().UnixNano()
	newV, err := item.GetValue()
	if err != nil {
		return err
	}
	return publishQueueWatcher.db.Set(newK, newV)
}

func (watcher *PublishQueueWatcher) GetGroupItems(groupId string, status string, trxId string) ([]*PublishQueueItem, error) {
	items := []*PublishQueueItem{}
	publishQueueWatcher.db.PrefixForeach([]byte(fmt.Sprintf("%s_%s", PUBQUEUE_PREFIX, groupId)), func(k []byte, v []byte, err error) error {
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
		valid := true
		if status != "" {
			valid = valid && (item.State == status)
		}
		if trxId != "" {
			valid = valid && (item.Trx.TrxId == trxId)
		}
		if valid {
			items = append(items, item)
		}

		return nil
	})
	return items, nil
}

func (watcher *PublishQueueWatcher) Ack(trxIds []string) ([]string, error) {
	acked := []string{}

	publishQueueWatcher.db.PrefixCondDelete([]byte(PUBQUEUE_PREFIX), func(k []byte, v []byte, err error) (bool, error) {
		if err != nil {
			chain_log.Warnf("<pubqueue>: %s", err.Error())
			// continue
			return false, err
		}
		item, err := ParsePublishQueueItem(k, v)
		if err != nil {
			chain_log.Warnf("<pubqueue>: %s", err.Error())
			return false, err
		}
		for _, trxId := range trxIds {
			if item.Trx.TrxId == trxId {
				acked = append(acked, trxId)
				return true, nil
			}
		}
		return false, nil
	})

	return acked, nil
}

var publishQueueWatcher PublishQueueWatcher = PublishQueueWatcher{}

func GetPubQueueWatcher() *PublishQueueWatcher {
	return &publishQueueWatcher
}

func InitPublishQueueWatcher(done chan bool, db storage.QuorumStorage) {

	publishQueueWatcher.db = db
	publishQueueWatcher.running = false

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
	if publishQueueWatcher.db == nil || publishQueueWatcher.running == true {
		return
	}
	publishQueueWatcher.running = true

	if autoAck {
		// auto remove
		publishQueueWatcher.db.PrefixCondDelete([]byte(PUBQUEUE_PREFIX), func(k []byte, v []byte, err error) (bool, error) {
			if err != nil {
				chain_log.Warnf("<pubqueue>: %s", err.Error())
				// continue
				return false, nil
			}
			item, err := ParsePublishQueueItem(k, v)
			if err != nil {
				chain_log.Warnf("<pubqueue>: %s", err.Error())
				return false, nil
			}
			if item.State == PublishQueueItemStateSuccess {
				return true, nil
			}

			if item.State == PublishQueueItemStateFail && item.RetryCount > MAX_RETRY_COUNT {
				return true, nil
			}

			return false, nil
		})
	}

	publishQueueWatcher.db.PrefixForeach([]byte(PUBQUEUE_PREFIX), func(k []byte, v []byte, err error) error {
		if err != nil {
			chain_log.Warnf("<pubqueue>: %s", err.Error())
			// continue
			return nil
		}

		// use goroutine to avoid thread blocking in browser(indexeddb doesn't support nested cursors)
		go func() {
			item, err := ParsePublishQueueItem(k, v)
			chain_log.Debugf("<pubqueue>: got item %v", item.Trx.TrxId)

			if err != nil {
				chain_log.Warnf("<pubqueue>: %s", err.Error())
				return
			}
			switch item.State {
			case PublishQueueItemStatePending:
				// check trx state, update, so it will be removed or retied in next poll
				groupmgr := GetGroupMgr()
				if group, ok := groupmgr.Groups[item.GroupId]; ok {
					// make sure data is updated to the latest change
					if group.GetSyncerStatus() != IDLE {
						chain_log.Debugf("<pubqueue>: group is not up to date yet.")
					}
					// try to find it from chain
					trx, _, err := group.GetTrx(item.Trx.TrxId)
					if err != nil {
						chain_log.Errorf("<pubqueue>: %s", err.Error())
						break
					}
					if trx.TrxId == item.Trx.TrxId {
						// synced
						chain_log.Debugf("<pubqueue>: trx %s success", trx.TrxId)
						item.State = PublishQueueItemStateSuccess
						break
					}
					// try to find it from cache
					trx, _, err = group.GetTrxFromCache(item.Trx.TrxId)
					if err != nil {
						chain_log.Errorf("<pubqueue>: %s", err.Error())
						break
					}
					if trx.TrxId == item.Trx.TrxId {
						chain_log.Debugf("<pubqueue>: trx %s success(from cache)", trx.TrxId)
						item.State = PublishQueueItemStateSuccess
						break
					}
					// failed or still pending, check the expire time
					chain_log.Debugf("<pubqueue>: trx %s not found, last updated at: %s, expire at: %s",
						item.Trx.TrxId,
						time.Unix(0, item.UpdateAt),
						time.Unix(0, item.Trx.Expired),
					)
					now := time.Now().UnixNano()
					if now >= item.Trx.Expired {
						// Failed
						chain_log.Infof("<pubqueue>: trx %s failed", item.Trx.TrxId)
						item.State = PublishQueueItemStateFail
					}

				}
			case PublishQueueItemStateFail:
				// retry then mark as pending
				groupmgr := GetGroupMgr()
				if group, ok := groupmgr.Groups[item.GroupId]; ok {
					if item.RetryCount > MAX_RETRY_COUNT {
						// TODO: this might consume some storage, not gonna clean it by now
						break
					}
					muser, ok := group.ChainCtx.Consensus.User().(*MolassesUser)
					if !ok {
						// ignore
						chain_log.Errorf("<pubqueue>: trx %s resend failed, cannot cast user node to MolassesUser", item.Trx.TrxId)
						break
					}
					trxId, err := muser.sendTrxWithoutRetry(item.Trx, conn.ProducerChannel)
					if err != nil {
						chain_log.Errorf("<pubqueue>: trx %s resend failed; error: %s", item.Trx.TrxId, err.Error())
					} else {
						updateTrxTimeLimit(item.Trx)
						item.State = PublishQueueItemStatePending
						item.RetryCount += 1

						chain_log.Debugf("<pubqueue>: trx %s resent(%d)", trxId, item.RetryCount)
					}

				}

			default:
			}

			err = publishQueueWatcher.UpsertItem(item)
			if err != nil {
				chain_log.Errorf("<pubqueue>: %s", err.Error())
			}
		}()
		return nil
	})
	publishQueueWatcher.running = false
}

func TrxEnqueue(groupId string, trx *quorumpb.Trx) error {
	//chain_log.Debugf("<pubqueue>: %v to group(%s)", trx.TrxId, groupId)
	item := PublishQueueItem{groupId, PublishQueueItemStatePending, 0, time.Now().UnixNano(), trx}
	return publishQueueWatcher.UpsertItem(&item)
}
