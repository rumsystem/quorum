package hbbft

import (
	"errors"
	"math/rand"
	"sync"
	"time"

	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

type buffer struct {
	lock        sync.RWMutex
	length      uint
	trxIdIdx    map[string]uint
	data        map[string]*quorumpb.Trx
	initialized bool
}

func New() *buffer {
	return &buffer{
		length:      0,
		trxIdIdx:    make(map[string]uint),
		data:        make(map[string]*quorumpb.Trx),
		initialized: false,
	}
}

func (b *buffer) Init() {
	b.lock.Lock()
	defer b.lock.Unlock()
	rand.Seed(time.Now().UnixNano())
	b.initialized = true
}

func (b *buffer) Push(trx *quorumpb.Trx) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if !b.initialized {
		return errors.New("buffer unininitialized")
	}

	if _, ok := b.trxIdIdx[trx.TrxId]; ok {
		return errors.New("trxId existed")
	}

	b.data[trx.TrxId] = trx
	b.trxIdIdx[trx.TrxId] = b.length
	b.length++
	return nil
}

func (b *buffer) Delete(trxId string) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if !b.initialized {
		return errors.New("buffer unininitialized")
	}

	if _, ok := b.trxIdIdx[trxId]; !ok {
		return errors.New("trxId not existed")
	}

	//get index of removed trx
	index := b.trxIdIdx[trxId]

	//update index after removed trx
	for k, v := range b.trxIdIdx {
		if v <= index {
			continue
		} else {
			b.trxIdIdx[k]--
		}
	}

	delete(b.data, trxId)
	delete(b.trxIdIdx, trxId)
	b.length--

	return nil
}

func (b *buffer) Len() uint {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.length
}

func (b *buffer) GetNRandSample(n uint) ([]*quorumpb.Trx, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if n > b.length || n <= 0 {
		return nil, errors.New("too much(less) sample, n should between 1 and buffer len")
	}

	return nil, nil
}
