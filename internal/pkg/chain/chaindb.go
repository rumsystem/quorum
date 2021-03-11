package chain

import (
	"encoding/json"
	//"fmt"
	"math/rand"
	"sync"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/golang/glog"
	"github.com/oklog/ulid"
)

//Save trx to local db
func AddTrx(trx Trx) error {
	err := GetContext().TrxDb.Update(func(txn *badger.Txn) error {
		bytes, err := json.Marshal(trx)
		e := badger.NewEntry([]byte(trx.Msg.TrxId), bytes)
		err = txn.SetEntry(e)
		return err
	})

	if err != nil {
		glog.Fatalf(err.Error())
	}

	return err
}

//Rm Trx
func RmTrx(trxId string) error {
	err := GetContext().TrxDb.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(trxId))
		return err
	})

	return err
}

func UpdTrxCons(trxId, consensusString string) error {
	if trx, err := GetTrx(trxId); err != nil {
		return err
	} else {
		trx.Consensus = append(trx.Consensus, consensusString)
		return AddTrx(trx)
	}
}

//Get trx
func GetTrx(trxId string) (Trx, error) {
	var trx Trx
	err := GetContext().TrxDb.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(trxId))

		if err != nil {
			return err
		}

		trxBytes, err := item.ValueCopy(nil)

		if err != nil {
			return err
		}

		err = json.Unmarshal(trxBytes, &trx)

		if err != nil {
			return err
		}

		return nil
	})

	return trx, err
}

//Save Block
func AddBlock(block Block, groupItem *GroupItem) error {

	//AddBlock to blockDb
	err := GetContext().BlockDb.Update(func(txn *badger.Txn) error {
		bytes, err := json.Marshal(block)
		e := badger.NewEntry([]byte(block.Cid), bytes)
		err = txn.SetEntry(e)
		return err
	})

	if err != nil {
		glog.Fatalf(err.Error())
	}

	return err
}

//Rm Block
func RmBlock(block Block) error {
	err := GetContext().BlockDb.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(block.Cid))
		return err
	})
	return err
}

//Upd Block
func UpdBlock(oldBlock, newBlock Block, groupItem *GroupItem) error {
	err := AddBlock(newBlock, groupItem)
	return err
}

//Get Block
func GetBlock(blockId string) (Block, error) {
	var block Block
	err := GetContext().BlockDb.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(blockId))
		blockBytes, err := item.ValueCopy(nil)

		if err != nil {
			return err
		}

		err = json.Unmarshal(blockBytes, &block)
		return err
	})

	return block, err
}

func GetRawBlock(blockId string) ([]byte, error) {
	var raw []byte
	err := GetContext().BlockDb.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(blockId))
		raw, err = item.ValueCopy(nil)
		return err
	})

	return raw, err
}

//for generate sequence ULID
type MonotonicULIDSource struct {
	sync.Mutex
	entropy  *rand.Rand
	lastMs   uint64
	lastULID ulid.ULID
}

func GenerateULID() ulid.ULID {
	entropy := rand.New(rand.NewSource(time.Unix(1000000, 0).UnixNano()))
	ulidSource := NewMonotonicULIDSource(entropy)
	now := time.Now()
	id, _ := ulidSource.New(now)

	return id
}

func NewMonotonicULIDSource(entropy *rand.Rand) *MonotonicULIDSource {
	initial, err := ulid.New(ulid.Now(), entropy)
	if err != nil {
		panic(err)
	}

	return &MonotonicULIDSource{
		entropy:  entropy,
		lastMs:   0,
		lastULID: initial,
	}
}

func (u *MonotonicULIDSource) New(t time.Time) (ulid.ULID, error) {
	u.Lock()
	defer u.Unlock()

	ms := ulid.Timestamp(t)
	var err error
	if ms > u.lastMs {
		u.lastMs = ms
		u.lastULID, err = ulid.New(ms, u.entropy)
		return u.lastULID, err
	}

	incrEntropy := incrementBytes(u.lastULID.Entropy())
	var dup ulid.ULID
	dup.SetTime(ms)

	if err := dup.SetEntropy(incrEntropy); err != nil {
		return dup, err
	}

	u.lastULID = dup
	u.lastMs = ms

	return dup, nil
}

func incrementBytes(in []byte) []byte {
	const (
		minByte byte = 0
		maxByte byte = 255
	)

	out := make([]byte, len(in))
	copy(out, in)

	leastSigByteIdx := len(out) - 1
	mostSigByteIdx := 0

	for i := leastSigByteIdx; i >= mostSigByteIdx; i-- {
		if out[i] == maxByte {
			out[i] = minByte
			continue
		}

		out[i]++
		return out
	}

	panic(ulid.ErrOverflow)
}
