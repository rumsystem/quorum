package taskqueue

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/dgraph-io/badger"
)

// queueOpts struct
type queueOpts struct {
	DBPath string
	Logger badger.Logger
}

// queue struct
type queue struct {
	opts queueOpts
	db   *badger.DB
	seq  *badger.Sequence
	dbL  sync.Mutex
}

// newQueue creates new ueue
func newQueue(opts queueOpts) *queue {
	q := &queue{opts: opts}
	return q
}

type badgerLogger struct{}

func (l *badgerLogger) Infof(format string, a ...interface{}) {
	// ignore badger info logs
}

func (l *badgerLogger) Errorf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
}

func (l *badgerLogger) Warningf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
}

func (l *badgerLogger) Debugf(format string, a ...interface{}) {
	// Leave blank for the time being,
	// track it in the future.
}

// start Queue
func (q *queue) start() error {
	// validate opts
	if q.opts.DBPath == "" {
		return errors.New("DBPath is required")
	}

	x := badger.Options{}

	x.Dir = "ds"

	// open db
	badgerOpts := badger.DefaultOptions(q.opts.DBPath)
	badgerOpts.Logger = &badgerLogger{}
	badgerOpts.SyncWrites = true

	db, err := badger.Open(badgerOpts)
	if err != nil {
		return err
	}
	q.db = db

	// init sequence
	q.seq, err = db.GetSequence([]byte("standard"), 1000)
	return err
}

// stop Queue and Release resources
func (q *queue) stop() error {
	// release sequence
	err := q.seq.Release()
	if err != nil {
		return err
	}

	// close db
	err = q.db.Close()
	if err != nil {
		return err
	}

	return nil
}

func getNextSeq(seq *badger.Sequence) (num uint64, err error) {
	defer func() {
		r := recover()
		if r != nil {
			// recover from panic and send err instead
			err = r.(error)
		}
	}()

	num, err = seq.Next()
	return num, err
}

// enqueueJob enqueues a new Job to the Pending queue
func (q *queue) enqueueJob(name string, data []byte) (uint64, error) {
	num, err := getNextSeq(q.seq)
	if err != nil {
		return 0, err
	}
	j := &Job{ID: num + 1, Name: name, Data: data}
	jKey := getJobKey(jobPending, j.ID)

	err = q.db.Update(func(txn *badger.Txn) error {
		b, err := encodeJob(j)
		if err != nil {
			return err
		}

		err = txn.Set([]byte(jKey), b)

		return err
	})
	if err != nil {
		return 0, err
	}

	return j.ID, nil
}

func encodeJob(j *Job) ([]byte, error) {
	var b bytes.Buffer
	err := gob.NewEncoder(&b).Encode(j)
	return b.Bytes(), err
}

// jobStatus Enum Type
type jobStatus uint8

const (
	// jobPending : waiting to be processed
	jobPending jobStatus = iota
	// jobInProgress : processing in progress
	jobInProgress
	// jobComplete : processing complete
	jobComplete
	// jobFailed : processing errored out
	jobFailed
)

func getQueueKeyPrefix(status jobStatus) string {
	return fmt.Sprintf("q:%v:", status)
}

func getJobKey(status jobStatus, jID uint64) string {
	return getQueueKeyPrefix(status) + jIDString(jID)
}

func jIDString(jID uint64) string {
	return fmt.Sprintf("%020d", jID)
}

func (q *queue) dequeueJob() (*Job, error) {
	var j *Job

	q.dbL.Lock()
	defer q.dbL.Unlock()
	err := q.db.Update(func(txn *badger.Txn) error {
		prefix := []byte(getQueueKeyPrefix(jobPending))
		k, v, err := getFirstKVForPrefix(txn, prefix)
		if err != nil {
			return err
		}
		// iteration is done, no job was found
		if k == nil {
			return nil
		}

		j, err = decodeJob(v)
		if err != nil {
			return err
		}

		// Move from from Pending queue to InProgress queue
		err = moveItem(txn, k, []byte(getJobKey(jobInProgress, j.ID)), v)

		return err
	})

	return j, err
}

func getFirstKVForPrefix(txn *badger.Txn, prefix []byte) ([]byte, []byte, error) {
	itOpts := badger.DefaultIteratorOptions
	itOpts.PrefetchValues = true
	itOpts.PrefetchSize = 1
	it := txn.NewIterator(itOpts)

	// go to smallest key after prefix
	it.Seek(prefix)
	defer it.Close()
	// iteration done, no item found
	if !it.ValidForPrefix(prefix) {
		return nil, nil, nil
	}

	item := it.Item()

	k := item.KeyCopy(nil)

	v, err := item.ValueCopy(nil)
	return k, v, err
}

// markJobDone moves a job from the inprogress status to complete/failed
func (q *queue) markJobDone(id uint64, status jobStatus) error {
	if status != jobComplete && status != jobFailed {
		return errors.New("Can only move to Complete or Failed Status")
	}

	key := []byte(getJobKey(jobInProgress, id))

	q.dbL.Lock()
	defer q.dbL.Unlock()
	err := q.db.Update(func(txn *badger.Txn) error {
		b, err := getBytesForKey(txn, key)
		if err != nil {
			return err
		}

		// Move from from InProgress queue to dest queue
		err = moveItem(txn, key, []byte(getJobKey(status, id)), b)

		return err
	})

	return err
}

func decodeJob(b []byte) (*Job, error) {
	var j *Job
	err := gob.NewDecoder(bytes.NewBuffer(b)).Decode(&j)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func getJobForKey(txn *badger.Txn, key []byte) (*Job, error) {
	b, err := getBytesForKey(txn, key)
	if err != nil {
		return nil, err
	}
	j, err := decodeJob(b)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func getBytesForKey(txn *badger.Txn, key []byte) ([]byte, error) {
	item, err := txn.Get(key)
	if err != nil {
		return nil, err
	}

	b, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func moveItem(txn *badger.Txn, oldKey []byte, newKey []byte, b []byte) error {
	// remove from Source queue
	err := txn.Delete(oldKey)
	if err != nil {
		return err
	}

	// create in Dest queue
	err = txn.Set(newKey, b)
	return err
}
