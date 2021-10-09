package model

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/rumsystem/quorum/cmd/cli/api"
	"github.com/rumsystem/quorum/cmd/cli/cache"
	"github.com/rumsystem/quorum/cmd/cli/config"
	qApi "github.com/rumsystem/quorum/internal/pkg/api"
)

var DefaultPagerOpt = api.PagerOpt{StartTrxId: "", Reverse: true, Page: 0}

// model
type QuorumDataModel struct {
	ForceUpdate   bool
	Pager         map[string]api.PagerOpt
	Users         map[string]api.ContentStruct
	Node          api.NodeInfoStruct
	Network       api.NetworkInfoStruct
	Groups        qApi.GroupInfoList
	Contents      []api.ContentStruct
	ContentFilter string
	// in memory cache
	Cache         map[string][]api.ContentStruct
	CurGroup      string
	RedrawCh      chan bool
	TickerCh      chan struct{}
	TickerRunning bool
	Counter       uint64

	sync.RWMutex
}

func (q *QuorumDataModel) SetForceUpdate(force bool) {
	q.RWMutex.Lock()
	defer q.RWMutex.Unlock()
	q.ForceUpdate = force
}

func (q *QuorumDataModel) StartTicker(fn func()) {
	q.RLock()
	TickerRunning := q.TickerRunning
	q.RUnlock()
	if !TickerRunning {
		Ticker := time.NewTicker(500 * time.Millisecond)
		q.RWMutex.Lock()
		q.TickerCh = make(chan struct{})
		q.RWMutex.Unlock()
		go func() {
			for {
				select {
				case <-Ticker.C:
					if api.IsValidApiServer() {
						fn()
						q.Counter += 1
					}
				case <-q.TickerCh:
					Ticker.Stop()
					q.RWMutex.Lock()
					q.TickerRunning = false
					q.RWMutex.Unlock()
					return
				}
			}
		}()
		q.RWMutex.Lock()
		q.TickerRunning = true
		q.RWMutex.Unlock()
	}
}
func (q *QuorumDataModel) StopTicker() {
	q.RLock()
	TickerRunning := q.TickerRunning
	q.RUnlock()
	if TickerRunning {
		q.RWMutex.Lock()
		close(q.TickerCh)
		q.RWMutex.Unlock()
	}
}

func (q *QuorumDataModel) GetUserProfile(pubkey string, groupId string) *api.ContentInnerProfileStruct {
	q.RLock()
	defer q.RUnlock()

	users := q.Users
	key := cache.GetUserProfileKey(groupId, pubkey)
	content, ok := users[key]
	if ok {
		profile := api.ContentInnerProfileStruct{}

		jsonStr, _ := json.Marshal(content.Content)
		json.Unmarshal(jsonStr, &profile)
		return &profile
	}
	go func() {
		// load from local, and update the profile
		c, _ := cache.QCache.Get([]byte(key))
		if c != nil {
			content := api.ContentStruct{}
			err := json.Unmarshal(c, &content)
			if err == nil {
				profile := api.ContentInnerProfileStruct{}
				jsonStr, _ := json.Marshal(content.Content)
				json.Unmarshal(jsonStr, &profile)

				q.UpdateUserInfo(content, groupId)
				config.Logger.Infof("Loaded user info from local: %s", profile.Name)
				q.RedrawCh <- true
			}
		}
	}()
	return nil
}

func (q *QuorumDataModel) GetUserName(pubkey string, groupId string) string {
	profile := q.GetUserProfile(pubkey, groupId)
	if profile != nil {
		return profile.Name
	}
	return pubkey
}

func (q *QuorumDataModel) GetUserMixinUID(pubkey string, groupId string) string {
	profile := q.GetUserProfile(pubkey, groupId)
	if profile != nil {
		wallets := profile.Wallet
		if len(wallets) > 0 {
			for _, w := range wallets {
				if w.Type == "mixin" {
					return w.Id
				}
			}
		}
	}
	return ""
}

func (q *QuorumDataModel) GetPager(groupId string) api.PagerOpt {
	q.RLock()
	defer q.RUnlock()

	pager := q.Pager

	page, ok := pager[groupId]
	if !ok {
		return DefaultPagerOpt
	}
	return page
}

func (q *QuorumDataModel) SetPager(groupId string, opt api.PagerOpt) {
	q.RWMutex.Lock()
	defer q.RWMutex.Unlock()

	pager := q.Pager
	pager[groupId] = opt
}

func (q *QuorumDataModel) UpdateUserInfo(content api.ContentStruct, groupId string) {
	q.RWMutex.Lock()
	defer q.RWMutex.Unlock()

	users := q.Users
	userPubKey := content.Publisher
	key := cache.GetUserProfileKey(groupId, userPubKey)
	// update in memory thing
	old, hasKey := users[key]
	if !hasKey {
		users[key] = content
	} else {
		if content.TimeStamp > old.TimeStamp {
			users[key] = content
		}
	}
	// update local db
	go func() {
		c, _ := cache.QCache.Get([]byte(key))
		if c == nil {
			val, err := json.Marshal(content)
			if err != nil {
				config.Logger.Errorf("Failed to update local cache: %s", err.Error())
				return
			}
			cache.QCache.Set([]byte(key), val)
		} else {
			dbContent := api.ContentStruct{}
			err := json.Unmarshal(c, &dbContent)
			if err == nil {
				if dbContent.TimeStamp < content.TimeStamp {
					val, err := json.Marshal(content)
					if err != nil {
						config.Logger.Errorf("Failed to update local cache: %s", err.Error())
						return
					}
					cache.QCache.Set([]byte(key), val)
				}
			}
		}
	}()
}

func (q *QuorumDataModel) SetNetworkInfo(network api.NetworkInfoStruct) {
	q.RWMutex.Lock()
	defer q.RWMutex.Unlock()

	q.Network = network
}

func (q *QuorumDataModel) GetNetworkInfo() api.NetworkInfoStruct {
	q.RLock()
	defer q.RUnlock()

	return q.Network
}

func (q *QuorumDataModel) SetNodeInfo(node api.NodeInfoStruct) {
	q.RWMutex.Lock()
	defer q.RWMutex.Unlock()

	q.Node = node
}

func (q *QuorumDataModel) GetNodeInfo() api.NodeInfoStruct {
	q.RLock()
	defer q.RUnlock()

	return q.Node
}

func (q *QuorumDataModel) SetGroups(groups qApi.GroupInfoList) {
	q.RWMutex.Lock()
	defer q.RWMutex.Unlock()

	q.Groups = groups
}

func (q *QuorumDataModel) GetGroups() qApi.GroupInfoList {
	q.RLock()
	defer q.RUnlock()

	return q.Groups
}

func (q *QuorumDataModel) GetContents() []api.ContentStruct {
	q.RLock()
	defer q.RUnlock()

	return q.Contents
}

func (q *QuorumDataModel) SetContents(contents []api.ContentStruct) {
	q.RWMutex.Lock()
	defer q.RWMutex.Unlock()

	q.Contents = contents
}

func (q *QuorumDataModel) GetCache(gid string) ([]api.ContentStruct, bool) {
	q.RLock()
	defer q.RUnlock()
	data, ok := q.Cache[gid]
	return data, ok
}

func (q *QuorumDataModel) UpdateCache(gid string, contents []api.ContentStruct) {
	q.RWMutex.Lock()
	defer q.RWMutex.Unlock()
	q.Cache[gid] = contents
}

func (q *QuorumDataModel) GetCurrentGroup() string {
	q.RLock()
	defer q.RUnlock()
	return q.CurGroup
}

func (q *QuorumDataModel) SetCurrentGroup(gid string) {
	q.RWMutex.Lock()
	defer q.RWMutex.Unlock()
	q.CurGroup = gid
}
