package model

import (
	"sync"
	"time"

	"github.com/rumsystem/quorum/cmd/cli/api"
)

type NetworkDataModel struct {
	Data          *map[string]api.PingInfoItemStruct
	CurPeer       string
	TickerCh      chan struct{}
	TickerRunning bool
	Counter       uint64

	sync.RWMutex
}

func (m *NetworkDataModel) StartTicker(fn func()) {
	m.RLock()
	TickerRunning := m.TickerRunning
	m.RUnlock()
	if !TickerRunning {
		Ticker := time.NewTicker(5000 * time.Millisecond)
		m.RWMutex.Lock()
		m.TickerCh = make(chan struct{})
		m.RWMutex.Unlock()
		go func() {
			for {
				select {
				case <-Ticker.C:
					if api.IsValidApiServer() {
						fn()
						m.Counter += 1
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
func (m *NetworkDataModel) StopTicker() {
	m.RLock()
	TickerRunning := m.TickerRunning
	m.RUnlock()
	if TickerRunning {
		m.RWMutex.Lock()
		close(m.TickerCh)
		m.RWMutex.Unlock()
	}
}

func (m *NetworkDataModel) SetData(data *map[string]api.PingInfoItemStruct) {
	m.RWMutex.Lock()
	defer m.RWMutex.Unlock()

	m.Data = data
}

func (m *NetworkDataModel) GetPeerData(peer string) *api.PingInfoItemStruct {
	m.RLock()
	defer m.RUnlock()

	data, ok := (*m.Data)[peer]
	if ok {
		return &data
	}
	return nil
}

func (m *NetworkDataModel) GetPeers() []string {
	m.RLock()
	defer m.RUnlock()
	peers := []string{}
	if m.Data == nil {
		return peers
	}
	for k := range *m.Data {
		peers = append(peers, k)
	}
	return peers
}

func (m *NetworkDataModel) GetCurrentPeer() string {
	m.RLock()
	defer m.RUnlock()
	peer := m.CurPeer
	return peer
}

func (m *NetworkDataModel) SetCurrentPeer(peer string) {
	m.RWMutex.Lock()
	defer m.RWMutex.Unlock()

	m.CurPeer = peer
}
