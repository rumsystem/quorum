package chain

import (
	"encoding/json"
	"fmt"
	"strings"

	"time"

	chaindef "github.com/rumsystem/quorum/internal/pkg/chainsdk/def"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type PublishQueueItem struct {
	// also stored in the key for quick indexing
	GroupId string `example:"5ed3f9fe-81e2-450d-9146-7a329aac2b62"`

	// in value only
	State      string `example:"SUCCESS"`
	RetryCount int    `example:"0"`
	UpdateAt   int64  `json:"UpdateAt,string" example:"1650786473293614500"`
	/* Trx Example:
		{
	        "TrxId": "b5433111-f3a1-41e2-a03f-648e47a04dad",
	        "GroupId": "6bd70de8-addc-4b03-8271-a5a5b02d1ebd",
	        "Data": "jvFEEhBuwRpu7or2IUt8NdTZ1R/qzlXeJeseU7csZi+XYC28Fufj3aORoKVCAXyxBxZCuHe7kp6tKAScNxClqEX82+As+fKsBK6zTpB9gyO+fn2y",
	        "TimeStamp": "1650532131665550100",
	        "Version": "1.0.0",
	        "Expired": 1650532161665550000,
	        "Nonce": 24000,
	        "SenderPubkey": "CAISIQMrNsVK8/ZrJylBFJZEe6BnslK7B5wAygbxde+RG9Hafg==",
	        "SenderSign": "MEQCIDZlG/ILNC89z/OYEuADqYpHfx81pqA3RnOlLSCeypP3AiAFKLSD8M8TyNr6quYFCnuL1nzMwUlHWiEiVimDFCHlmQ=="
	      }
	*/
	Trx         *quorumpb.Trx
	StorageType string `example:"CHAIN"`
}

const (
	PublishQueueItemStatePending = "PENDING"
	PublishQueueItemStateSuccess = "SUCCESS"
	PublishQueueItemStateFail    = "FAIL"
	PUBQUEUE_PREFIX              = "PUBQUEUE"
	MAX_RETRY_COUNT              = 10
)

var (
	autoAck bool = false
)

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
	db            storage.QuorumStorage
	groupMgrIface chaindef.GroupMgrIface
	running       bool
}

func (watcher *PublishQueueWatcher) UpsertItem(item *PublishQueueItem) error {
	if watcher.db == nil {
		chain_log.Error("watcher.db is nil")
		return nil
	}
	newK, err := item.GetKey()
	if err != nil {
		chain_log.Error("can not get db key for item")
		return err
	}
	item.UpdateAt = time.Now().UnixNano()
	newV, err := item.GetValue()
	if err != nil {
		chain_log.Error("can not get db value for item")
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

func InitPublishQueueWatcher(done chan bool, groupMgrIface chaindef.GroupMgrIface, db storage.QuorumStorage) {
	publishQueueWatcher.db = db
	publishQueueWatcher.running = false
	publishQueueWatcher.groupMgrIface = groupMgrIface
	if db != nil {
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
		item, err := ParsePublishQueueItem(k, v)
		if err != nil {
			chain_log.Warnf("<pubqueue>: %s", err)
			return err
		}

		//chain_log.Debugf("<pubqueue>: got item %v group_id %s", item.Trx.TrxId, item.GroupId)
		go func() {

			switch item.State {
			case PublishQueueItemStatePending:
				// check trx state, update, so it will be removed or retied in next poll
				group, err := publishQueueWatcher.groupMgrIface.GetGroup(item.GroupId)
				if err == nil {
					// make sure data is updated to the latest change
					// commented by cuicat
					/*
						if group.GetSyncerStatus() != IDLE {
							chain_log.Debugf("<pubqueue>: group is not up to date yet.")
						}
					*/
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
						item.StorageType = trx.StorageType.String()
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
						item.StorageType = trx.StorageType.String()
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
				if item.RetryCount > MAX_RETRY_COUNT {
					// TODO: this might consume some storage, not gonna clean it by now
					break
				}

				connMgr, err := conn.GetConn().GetConnMgr(item.GroupId)
				if err == nil {
					err := connMgr.SendUserTrxPubsub(item.Trx)
					if err != nil {
						chain_log.Errorf("<pubqueue>: trx %s resend failed; error: %s", item.Trx.TrxId, err.Error())
					} else {
						//FIX: don't update trx time
						//rumchaindata.UpdateTrxTimeLimit(item.Trx)
						item.State = PublishQueueItemStatePending
						item.RetryCount += 1

						chain_log.Debugf("<pubqueue>: trx %s resent(%d)", item.Trx.TrxId, item.RetryCount)
					}
				} else {
					chain_log.Errorf("<pubqueue>: trx %s resend failed; error: %s", item.Trx.TrxId, err.Error())
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

func (watcher *PublishQueueWatcher) TrxEnqueue(groupId string, trx *quorumpb.Trx) error {
	chain_log.Debugf("<TrxEnqueue>: %s to group %s", trx.TrxId, groupId)
	item := PublishQueueItem{groupId, PublishQueueItemStatePending, 0, time.Now().UnixNano(), trx, ""}
	return watcher.UpsertItem(&item)
}

func TrxEnqueue(groupId string, trx *quorumpb.Trx) error {
	return publishQueueWatcher.TrxEnqueue(groupId, trx)
}
