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

type BlockSyncTask struct {
	BlockId   string
	Direction Syncdirection
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
	Status           int8
	waitResultTaskId string //waiting the result for taskid
	retryCount       int8
	retrycountmu     sync.Mutex
	taskq            chan *SyncTask
	resultq          chan *SyncResult
	retrynext        bool //workaround for rumexchange
	taskdone         chan struct{}
	stopnotify       chan struct{}

	nextTask       func(taskid string) (*SyncTask, error)   //request the next task
	resultreceiver func(result *SyncResult) (string, error) //receive resutls and process them (save to the db, update chain...), return an id related with next task and error
	tasksender     func(task *SyncTask) error               //send task via network or others
}

func NewGsyncer(groupid string, getTask func(taskid string) (*SyncTask, error), resultreceiver func(result *SyncResult) (string, error), tasksender func(task *SyncTask) error) *Gsyncer {
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
	safeClose(s.taskdone)
	safeCloseTask(s.taskq)
	safeCloseResult(s.resultq)
	if s.stopnotify != nil {
		signcount := 0
		for _ = range s.stopnotify {
			signcount++
			//wait stop sign and set idle
			if signcount == 2 { // taskq and resultq stopped
				s.Status = IDLE
				close(s.stopnotify)
			}
		}
	}
}

func (s *Gsyncer) Start() {
	s.taskq = make(chan *SyncTask)
	s.resultq = make(chan *SyncResult, 3)
	s.taskdone = make(chan struct{})
	s.stopnotify = make(chan struct{})
	gsyncer_log.Debugf("<%s> Gsyncer Start", s.GroupId)
	go func() {
		for task := range s.taskq {
			ctx, cancel := context.WithTimeout(context.Background(), 2*RESULT_TIMEOUT*time.Second)
			defer cancel()
			err := s.processTask(ctx, task)
			if err == nil {
				gsyncer_log.Debugf("<%s> process task done %s", s.GroupId, task.Id)
			} else {
				//retry this task
				if s.retrynext == true {
					gsyncer_log.Debugf("<%s> task process %s error: %s, retry with next...", s.GroupId, task.Id, err)
					nexttask, err := s.nextTask("0")
					if err == nil {
						s.AddTask(nexttask)
					}
				} else {
					gsyncer_log.Debugf("<%s> task process %s error: %s, retry...", s.GroupId, task.Id, err)
					s.RetryCounterInc()
					s.AddTask(task)
				}
			}
		}
		s.stopnotify <- struct{}{}
	}()

	go func() {
		for result := range s.resultq {
			ctx, cancel := context.WithTimeout(context.Background(), RESULT_TIMEOUT*time.Second)
			defer cancel()
			nexttaskid, err := s.processResult(ctx, result)
			if err == nil {
				//test try to add next task
				gsyncer_log.Debugf("<%s> process result done %s", s.GroupId, result.Id)
				if nexttaskid == "" {
					gsyncer_log.Debugf("nextTask can't be null, skip")
					continue
				}

				nexttask, err := s.nextTask(nexttaskid)
				if err != nil {
					gsyncer_log.Debugf("nextTask error:%s", err)
					continue
				}
				s.AddTask(nexttask)
			} else if err == ErrSyncDone {
				gsyncer_log.Infof("<%s> result %s is Sync Pause Signal", s.GroupId, result.Id)
				//SyncPause, stop add next task, pause
			} else if err == ErrNoTaskWait {
				//no task waiting, skip don't add new task
			} else if err == ErrNotAccept {
				//not accept, do nothing and waiting for other reponses until timeout
			} else if err == ErrIgnore {
				//ignore, don't add next task, and waiting for timeout and process next
			} else {
				nexttask, terr := s.nextTask("") //the taskid shoule be inclued in the result, which need to upgrade all publicnode. so a workaround, pass a "" to get a retry task. (runner will try to maintain a taskid)
				if terr != nil {
					gsyncer_log.Debugf("nextTask error:%s", terr)
					continue
				}
				//test try to retry this task
				s.AddTask(nexttask)
				gsyncer_log.Debugf("<%s> result process %s error: %s", s.GroupId, result.Id, err)
			}
		}
		s.stopnotify <- struct{}{}
	}()
}

func (s *Gsyncer) processResult(ctx context.Context, result *SyncResult) (string, error) {
	resultdone := make(chan struct{})
	var err error
	var nexttaskid string
	go func() {
		nexttaskid, err = s.resultreceiver(result)
		select {
		case resultdone <- struct{}{}:
			gsyncer_log.Debugf("<%s> done result: %s", s.GroupId, result.Id)
		default:
			return
		}
	}()

	select {
	case <-resultdone:
		if s.waitResultTaskId != "" {
			//for TEST fake workload
			//time.Sleep(10 * time.Second)
			if err != ErrIgnore { // ErrIgnore will let the task wait for timeout, instead of be terminated
				s.taskdone <- struct{}{}
			}
			return nexttaskid, err
		} else {
			return nexttaskid, ErrNoTaskWait
		}
	case <-ctx.Done():
		return "", errors.New("Result Timeout")
	}
}

func (s *Gsyncer) processTask(ctx context.Context, task *SyncTask) error {
	//TODO: close this goroutine when the processTask func return. add some defer signal?
	go func() {
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

func (s *Gsyncer) AddTask(task *SyncTask) {
	go func() {
		s.taskq <- task
	}()
}

func (s *Gsyncer) AddResult(result *SyncResult) {
	go func() {
		s.resultq <- result
	}()
}
