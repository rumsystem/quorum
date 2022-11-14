package chain

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/logging"
)

var gsyncer_log = logging.Logger("syncer")

var (
	ErrSyncDone   = errors.New("Error Signal Sync Done")
	ErrNoTaskWait = errors.New("Error No Task Waiting Result")
	ErrNotAccept  = errors.New("Error The Result had been rejected")
	ErrIgnore     = errors.New("Ignore and wait for time out")
)

const RESULT_TIMEOUT = 4 //seconds

type Syncdirection uint

const (
	Next Syncdirection = iota
	Previous
)

type EpochSyncTask struct {
	Epoch int64
}

type SyncTask struct {
	Meta interface{}
	Id   string
}

type TimeoutNoResult struct {
	TaskId string
}

type SyncResult struct {
	Data   interface{}
	Id     string
	TaskId string
}

type Gsyncer struct {
	nodeName         string
	GroupId          string
	waitEpoch        int64 //waiting the task response for epoch
	Status           int8
	waitResultTaskId string //waiting the result for taskid
	retryCount       int8
	retrycountmu     sync.Mutex
	taskq            chan *SyncTask
	resultq          chan *SyncResult
	retrynext        bool //workaround for rumexchange
	taskdone         chan struct{}
	stopnotify       chan struct{}

	nextTask       func(epoch int64) (*SyncTask, error)    //request the next task
	resultreceiver func(result *SyncResult) (int64, error) //receive resutls and process them (save to the db, update chain...), return an id related with next task and error
	tasksender     func(task *SyncTask) error              //send task via network or others
}

func NewGsyncer(groupid string, getTask func(epoch int64) (*SyncTask, error), resultreceiver func(result *SyncResult) (int64, error), tasksender func(task *SyncTask) error) *Gsyncer {
	gsyncer_log.Debugf("<%s> NewGsyncer called", groupid)
	s := &Gsyncer{}
	s.Status = IDLE
	s.GroupId = groupid
	s.nextTask = getTask
	s.resultreceiver = resultreceiver
	s.tasksender = tasksender
	s.retryCount = 0

	return s
}

func (s *Gsyncer) GetWaitEpoch() int64 {
	return s.waitEpoch
}

func (s *Gsyncer) SetRetryWithNext(retrynext bool) {
	s.retrynext = retrynext
}

func (s *Gsyncer) RetryCounterInc() {
	s.retrycountmu.Lock()
	s.retryCount++
	s.retrycountmu.Unlock()
}
func (s *Gsyncer) RetryCounterClear() {
	s.retrycountmu.Lock()
	s.retryCount = 0
	s.retrycountmu.Unlock()
}

func (s *Gsyncer) RetryCounter() int8 {
	return s.retryCount
}

func safeClose(ch chan struct{}) (recovered bool) {
	defer func() {
		if recover() != nil {
			recovered = true
		}
	}()
	if ch == nil {
		return false
	}
	close(ch)
	return false
}
func safeCloseTask(ch chan *SyncTask) (recovered bool) {
	defer func() {
		if recover() != nil {
			recovered = true
		}
	}()
	if ch == nil {
		return false
	}
	close(ch)
	return false
}
func safeCloseResult(ch chan *SyncResult) (recovered bool) {
	defer func() {
		if recover() != nil {
			recovered = true
		}
	}()
	if ch == nil {
		return false
	}
	close(ch)
	return false
}

func (s *Gsyncer) Stop() {
	chain_log.Infof("syncer <%s> stopping...", s.GroupId)
	s.Status = CLOSE
	safeCloseTask(s.taskq)
	safeCloseResult(s.resultq)
	safeClose(s.taskdone)
	if s.stopnotify != nil {
		signcount := 0
		for _ = range s.stopnotify {
			signcount++
			//wait stop sign and set idle
			if signcount == 2 { // taskq and resultq stopped
				s.Status = IDLE
				close(s.stopnotify)
				chain_log.Infof("syncer <%s> stop success.", s.GroupId)
			}
		}
	}
}

func (s *Gsyncer) Start() {
	s.taskq = make(chan *SyncTask)
	s.resultq = make(chan *SyncResult, 3)
	s.taskdone = make(chan struct{})
	s.stopnotify = make(chan struct{})
	s.Status = SYNCING_FORWARD
	gsyncer_log.Debugf("<%s> Gsyncer Start", s.GroupId)
	go func() {
		for task := range s.taskq {
			ctx, cancel := context.WithTimeout(context.Background(), 2*RESULT_TIMEOUT*time.Second)
			defer cancel()
			err := s.processTask(ctx, task)
			if err == nil {
				gsyncer_log.Debugf("<%s> process task done %s", s.GroupId, task.Id)
			} else {
				//NO NEED RETRY for get epoch task
				//retry this task
				//if s.retrynext == true {
				//	gsyncer_log.Debugf("<%s> task process %s error: %s, retry with next...", s.GroupId, task.Id, err)
				//	nexttask, err := s.nextTask()
				//	if err == nil {
				//		s.AddTask(nexttask)
				//	}
				//} else {
				gsyncer_log.Debugf("<%s> task process %s error: %s, retry...", s.GroupId, task.Id, err)
				s.RetryCounterInc()
				s.addTask(task)
				//}
			}
		}
		s.stopnotify <- struct{}{}
	}()

	go func() {
		for result := range s.resultq {
			ctx, cancel := context.WithTimeout(context.Background(), RESULT_TIMEOUT*time.Second)
			defer cancel()
			nextepoch, err := s.processResult(ctx, result)
			if err == nil {
				//test try to add next task
				gsyncer_log.Debugf("<%s> process result done %s", s.GroupId, result.Id)
				if nextepoch == 0 {
					gsyncer_log.Debugf("nextTask can't be null, skip")
					continue
				}

				nexttask, err := s.nextTask(nextepoch)
				if err != nil {
					gsyncer_log.Debugf("nextTask error:%s", err)
					continue
				}
				s.addTask(nexttask)
			} else if err == ErrSyncDone {
				gsyncer_log.Infof("<%s> result %s is Sync Pause Signal", s.GroupId, result.Id)
				//SyncPause, stop add next task, pause
			}
		}
		s.stopnotify <- struct{}{}
	}()
}

func (s *Gsyncer) processResult(ctx context.Context, result *SyncResult) (int64, error) {
	resultdone := make(chan struct{})
	var err error
	var nextepoch int64
	go func() {
		nextepoch, err = s.resultreceiver(result)
		select {
		case resultdone <- struct{}{}:
			gsyncer_log.Debugf("<%s> done result: %s", s.GroupId, result.Id)
		default:
			return
		}
	}()

	select {
	case <-resultdone:
		gsyncer_log.Debugf("<%s> processResult done, waitEpoch %d nextepoch %d", s.GroupId, s.waitEpoch, nextepoch)
		if err == nil { //success
			if s.waitEpoch > 0 && s.waitEpoch == nextepoch-1 {
				//clean the wait epoch
				s.waitEpoch = 0
				s.retryCount = 0
				s.taskdone <- struct{}{}
				return nextepoch, err
			}
			gsyncer_log.Debugf("<%s> processResult done, ignore.", s.GroupId)
			return 0, ErrIgnore // ignore
		}
		gsyncer_log.Debugf("<%s> processResult done, ignore.", s.GroupId)
		return 0, err // ignore
	case <-ctx.Done():
		return 0, errors.New("Result Timeout")
	}
}

func (s *Gsyncer) processTask(ctx context.Context, task *SyncTask) error {
	//TODO: close this goroutine when the processTask func return. add some defer signal?
	go func() {
		//set waiting epoch
		//Epoch
		epochtask, ok := task.Meta.(EpochSyncTask)
		if ok == true {
			s.waitEpoch = epochtask.Epoch
		}
		s.waitResultTaskId = task.Id //set waiting task
		s.tasksender(task)
		//TODO: lock
	}()

	select {
	case <-s.taskdone:
		gsyncer_log.Debugf("<%s> receive taskdone event, clean waitResultTaskId", s.GroupId)
		s.waitResultTaskId = ""
		return nil
	case <-ctx.Done():
		s.waitResultTaskId = ""
		return errors.New("Task Timeout")
	}
}

func (s *Gsyncer) addTask(task *SyncTask) {
	go func() {
		if s.Status != CLOSE {
			s.taskq <- task
		}
	}()
}

func (s *Gsyncer) AddResult(result *SyncResult) {
	go func() {
		if s.Status != CLOSE {
			s.resultq <- result
		}
	}()
}
