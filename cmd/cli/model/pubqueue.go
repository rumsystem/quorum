package model

import (
	"sync"
	"time"

	"github.com/rumsystem/quorum/cmd/cli/api"
	qApi "github.com/rumsystem/quorum/internal/pkg/chainsdk/api"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
)

type PubqueueDataModel struct {
	Groups        qApi.GroupInfoList
	Trxs          []*chain.PublishQueueItem
	Cache         map[string][]*chain.PublishQueueItem
	CurGroup      string
	TickerCh      chan struct{}
	TickerRunning bool

	sync.RWMutex
}

func (m *PubqueueDataModel) StartTicker(fn func()) {
	m.RLock()
	TickerRunning := m.TickerRunning
	m.RUnlock()
	if !TickerRunning {
		Ticker := time.NewTicker(500 * time.Millisecond)
		m.RWMutex.Lock()
		m.TickerCh = make(chan struct{})
		m.RWMutex.Unlock()
		go func() {
			for {
				select {
				case <-Ticker.C:
					if api.IsValidApiServer() {
						fn()
					}
				case <-m.TickerCh:
					Ticker.Stop()
					m.RWMutex.Lock()
					m.TickerRunning = false
					m.RWMutex.Unlock()
					return
				}
			}
		}()
		m.RWMutex.Lock()
		m.TickerRunning = true
		m.RWMutex.Unlock()
	}
}
func (m *PubqueueDataModel) StopTicker() {
	m.RLock()
	TickerRunning := m.TickerRunning
	m.RUnlock()
	if TickerRunning {
		m.RWMutex.Lock()
		close(m.TickerCh)
		m.RWMutex.Unlock()
	}
}

func (m *PubqueueDataModel) SetGroups(groups qApi.GroupInfoList) {
	m.RWMutex.Lock()
	defer m.RWMutex.Unlock()

	m.Groups = groups
}

func (m *PubqueueDataModel) GetGroups() qApi.GroupInfoList {
	m.RLock()
	defer m.RUnlock()

	return m.Groups
}

func (m *PubqueueDataModel) GetCurrentGroup() string {
	m.RLock()
	defer m.RUnlock()
	return m.CurGroup
}

func (m *PubqueueDataModel) SetCurrentGroup(gid string) {
	m.RWMutex.Lock()
	defer m.RWMutex.Unlock()
	m.CurGroup = gid
}

func (m *PubqueueDataModel) GetTrx() []*chain.PublishQueueItem {
	m.RLock()
	defer m.RUnlock()

	return m.Trxs
}

func (m *PubqueueDataModel) SetTrxs(trxs []*chain.PublishQueueItem) {
	m.RWMutex.Lock()
	defer m.RWMutex.Unlock()

	m.Trxs = trxs
}

func (m *PubqueueDataModel) GetCache(gid string) ([]*chain.PublishQueueItem, bool) {
	m.RLock()
	defer m.RUnlock()
	data, ok := m.Cache[gid]
	return data, ok
}

func (m *PubqueueDataModel) UpdateCache(gid string, contents []*chain.PublishQueueItem) {
	m.RWMutex.Lock()
	defer m.RWMutex.Unlock()
	m.Cache[gid] = contents
}
