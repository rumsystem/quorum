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
	ErrSyncDone     = errors.New("Error Signal Sync Done")
	ErrNotAskedByMe = errors.New("Error Get Sync Resp but not asked by me")
	ErrNoTaskWait   = errors.New("Error No Task Waiting Result")
	ErrNotAccept    = errors.New("Error The Result had been rejected")
	ErrIgnore       = errors.New("Ignore")
	//ErrIgnore        = errors.New("Ignore and wait for time out")
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

type ConsensusSyncTask struct {
	SessionId string
}

type TaskType uint

const (
	GetEpoch TaskType = iota
	ConsensusSync
)

type SyncTask struct {
	TaskId string
	Type   TaskType
	Meta   interface{}
}

type SyncResult struct {
	TaskId string
	Type   TaskType
}

type Gsyncer struct {
	GroupId string
	Status  int8

	CurrentTask  *SyncTask
	retryCount   int8
	retrycountmu sync.Mutex

	//chan signals
	taskq      chan *SyncTask
	resultq    chan *SyncResult
	taskdone   chan struct{}
	stopnotify chan struct{}

	taskGenerators map[TaskType]func(args ...interface{}) (*SyncTask, error)
	resultHandlers map[TaskType]func(result *SyncResult) (string, error)
	tasksender     func(task *SyncTask) error //send task via network or others

	retrynext bool //workaround for rumexchange
	//nodeName         string
	//nextTask       func(epoch int64) (*SyncTask, error)    //request the next task
}

func NewGsyncer(groupid string,
	taskGenerators map[TaskType]func(args ...interface{}) (*SyncTask, error),
	resultHandlers map[TaskType]func(result *SyncResult) (string, error),
	tasksender func(task *SyncTask) error) *Gsyncer {
	gsyncer_log.Debugf("<%s> NewGsyncer called", groupid)

	s := &Gsyncer{}

	s.GroupId = groupid
	s.Status = IDLE

	s.retryCount = 0

	s.taskGenerators = taskGenerators
	s.resultHandlers = resultHandlers
	s.tasksender = tasksender

	//s.nextTask = getTask
	return s
}

func (s *Gsyncer) GetCurrentTask() (string, TaskType, error) {
	if s.CurrentTask == nil {
		return "", 0, ErrNoTaskWait
	}
	return s.CurrentTask.TaskId, s.CurrentTask.Type, nil
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

func (s *Gsyncer) GetRetryCount() int8 {
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
	gsyncer_log.Debugf("<%s> GSyncer stopping...", s.GroupId)
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
				gsyncer_log.Debugf("gsyncer <%s> stop success.", s.GroupId)
			}
		}
	}
}

func (s *Gsyncer) Start() {
	gsyncer_log.Debugf("<%s> Gsyncer Start", s.GroupId)
	s.taskq = make(chan *SyncTask)
	s.resultq = make(chan *SyncResult, 3)
	s.taskdone = make(chan struct{})
	s.stopnotify = make(chan struct{})

	//taskq
	go func() {
		for task := range s.taskq {
			ctx, cancel := context.WithTimeout(context.Background(), 2*RESULT_TIMEOUT*time.Second)
			defer cancel()
			err := s.processTask(ctx, task)
			if err == nil {
				gsyncer_log.Debugf("<%s> process task <%s> done", s.GroupId, task.TaskId)
			} else {
				gsyncer_log.Debugf("<%s> task <%s> process error: %s, retry...", s.GroupId, task.TaskId, err)
				s.RetryCounterInc()
				s.AddTask(task)
			}
		}
		s.stopnotify <- struct{}{}
	}()

	//resultq
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

				//TBD add new task according to different scenes
				//nexttask, err := s.GetATask[] nextTask(nextepoch)
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
	var taskId string
	go func() {
		taskId, err = s.resultHandlers[result.Type](result)
		select {
		case resultdone <- struct{}{}:
			gsyncer_log.Debugf("<%s> done result: %s", s.GroupId, result.TaskId)
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
				s.RetryCounterClear() //reset retry counter
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
	gsyncer_log.Debugf("processTask called")

	go func() {
		s.CurrentTask = task //set current task
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
	gsyncer_log.Debugf("Gsyncer addTask called")
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

/*


	//if curChainCnf > myEpoch, start sync
	if resp.CurChainEpoch > chain.group.Item.Epoch {
		chain_log.Debugf("Miss something, start sync")
		chain.syncerrunner.()
	} else {
		chain_log.Debugf("same epoch with chain, no need to sync, do nothing")
	}

*/
