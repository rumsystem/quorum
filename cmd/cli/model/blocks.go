package model

import (
	"sync"
	"time"

	"github.com/rumsystem/quorum/cmd/cli/api"
	qApi "github.com/rumsystem/quorum/internal/pkg/api"
)

// can only check from back to front
type BlockRangeOpt struct {
	CurBlockId  string
	NextBlockId string
	Count       int
	Done        bool
}

var DefaultBlockRange = BlockRangeOpt{"", "", 20, false}

type BlocksDataModel struct {
	Pager         map[string]BlockRangeOpt
	Groups        qApi.GroupInfoList
	Blocks        []api.BlockStruct
	NextBlocks    map[string][]string
	Cache         map[string][]api.BlockStruct
	CurGroup      string
	TickerCh      chan struct{}
	TickerRunning bool
	Counter       uint64

	sync.RWMutex
}

func (m *BlocksDataModel) StartTicker(fn func()) {
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
func (m *BlocksDataModel) StopTicker() {
	m.RLock()
	TickerRunning := m.TickerRunning
	m.RUnlock()
	if TickerRunning {
		m.RWMutex.Lock()
		close(m.TickerCh)
		m.RWMutex.Unlock()
	}
}

func (m *BlocksDataModel) SetGroups(groups qApi.GroupInfoList) {
	m.RWMutex.Lock()
	defer m.RWMutex.Unlock()

	m.Groups = groups
}

func (m *BlocksDataModel) GetGroups() qApi.GroupInfoList {
	m.RLock()
	defer m.RUnlock()

	return m.Groups
}

func (m *BlocksDataModel) GetCache(gid string) ([]api.BlockStruct, bool) {
	m.RLock()
	defer m.RUnlock()
	data, ok := m.Cache[gid]
	return data, ok
}

func (m *BlocksDataModel) UpdateCache(gid string, contents []api.BlockStruct) {
	m.RWMutex.Lock()
	defer m.RWMutex.Unlock()
	m.Cache[gid] = contents
}

func (m *BlocksDataModel) GetCurrentGroup() string {
	m.RLock()
	defer m.RUnlock()
	return m.CurGroup
}

func (m *BlocksDataModel) SetCurrentGroup(gid string) {
	m.RWMutex.Lock()
	defer m.RWMutex.Unlock()
	m.CurGroup = gid
}

func (m *BlocksDataModel) GetBlocks() []api.BlockStruct {
	m.RLock()
	defer m.RUnlock()

	return m.Blocks
}

func (m *BlocksDataModel) GetBlockById(id string) *api.BlockStruct {
	m.RLock()
	defer m.RUnlock()

	blocks := m.Blocks
	for _, block := range blocks {
		if block.BlockId == id {
			return &block
		}
	}
	return nil
}

func (m *BlocksDataModel) GetBlockByIndex(i int) *api.BlockStruct {
	m.RLock()
	defer m.RUnlock()

	blocks := m.Blocks
	if i < len(blocks) {
		return &blocks[i]
	}
	return nil
}

func (m *BlocksDataModel) SetBlocks(blocks []api.BlockStruct) {
	m.RWMutex.Lock()
	defer m.RWMutex.Unlock()

	m.Blocks = blocks
}

func (m *BlocksDataModel) GetPager(groupId string) BlockRangeOpt {
	m.RLock()
	defer m.RUnlock()

	pager := m.Pager

	page, ok := pager[groupId]
	if !ok {
		return DefaultBlockRange
	}
	return page
}

func (m *BlocksDataModel) SetPager(groupId string, opt BlockRangeOpt) {
	m.RWMutex.Lock()
	defer m.RWMutex.Unlock()

	pager := m.Pager
	pager[groupId] = opt
}

func (m *BlocksDataModel) SetNextBlock(prev string, next string) int {
	m.RWMutex.Lock()
	defer m.RWMutex.Unlock()

	nbmap := m.NextBlocks

	blocks, ok := nbmap[prev]
	if ok && len(blocks) > 0 {
		for _, item := range blocks {
			if item == next {
				return len(blocks)
			}
		}
		blocks = append(blocks, next)
		nbmap[prev] = blocks
		return len(blocks)
	} else {
		nextBlocks := []string{}
		nextBlocks = append(nextBlocks, next)
		nbmap[prev] = nextBlocks
		return 1
	}
}

func (m *BlocksDataModel) GetNextBlocks(blockId string) []string {
	m.RLock()
	defer m.RUnlock()
	ret := []string{}

	ret, ok := m.NextBlocks[blockId]
	if !ok {
		return ret
	}
	return ret

}
