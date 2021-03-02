package chain

import (
	//"fmt"
	//"encoding/json"
	"math/rand"
	"sync"
	"time"

	//"github.com/golang/glog"
	//badger "github.com/dgraph-io/badger/v3"
	"github.com/oklog/ulid"
)

//Save trx to local db
func AddTrx(trx Trx) error {
	/*
		txn := db.NewTransaction(true)
		if eSerial, err := json.Marshal(block); err != nil {
			return err
		} else if binCID, err := blockID.MarshalBinary(); err != nil {
			return err
		} else if err := txn.Set(binCID, []byte(eSerial)); err != nil {
			return err
		} else if err := txn.Commit(); err != nil {
			return err
		} else {
			return nil
		}
	*/

	//test only
	//Should add to trx.db

	GetContext().TrxItem[trx.Msg.TrxId] = trx
	return nil
}

//Rm Trx
func RmTrx(trxid string) error {

	//test only
	//Should rm from trx.db

	delete(GetContext().TrxItem, trxid)
	return nil
}

//Update Trx
//func UpdTrx(oldTrx, newTrx Trx) error {
//}

func UpdTrxCons(trxId, witnessSign string) error {

	//test only
	//should upd trx inside trx.db
	trx, _ := GetContext().TrxItem[trxId]
	trx.Consensus = append(trx.Consensus, witnessSign)
	GetContext().TrxItem[trx.Msg.TrxId] = trx
	return nil
}

//Get trx
func GetTrx(trxId string) (Trx, error) {

	//test only
	//should get from trx.db
	trx, ok := GetContext().TrxItem[trxId]

	if ok {
		return trx, nil
	} else {
		//return an error here
		return trx, nil
	}
}

//Save Block
func AddBlock(block Block) error {
	return nil
}

//Rm Block
func RmBlock(block Block) error {
	return nil
}

//Upd Block
func UpdBlock(oldBlock, newBlock Block) error {
	return nil
}

//Get Block
func GetBlock(blockId string) (Block, error) {
	var block Block
	return block, nil
}

//Get top block of a group
func GetTopBlock(groupId string) (Block, error) {

	testGroup, _ := GetContext().Group[TestGroupId]
	//check testGroup exist

	topBlock := testGroup.BlocksList[len(testGroup.BlocksList)-1]
	//check blockList not empty
	return topBlock, nil
}

/*
func GetItem(blockId ulid.ULID) (Block, error) {

	var block Block
	return block, nil

		if binCID, err := cid.MarshalBinary(); err != nil {
			return block, err
		}


			err := db.View(func(txn *badger.Txn) error {
				item, err := txn.Get(binCID)

				if (err != nil) {
					return block, err
				}

				var valCopy []byte

				err := item.Value(func(val []byte) error{
					valCopy = append([]byte{}, val...)
					return nil
				if (err != nil)	{
					return err
				}

				err := json.Unmarshal(valCopy, &block)

				if (err != nil) {
					return err
				}
			})
}
*/

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
