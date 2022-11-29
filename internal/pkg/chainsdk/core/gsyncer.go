package chain

import (
	"context"
	"errors"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/logging"
)

var gsyncer_log = logging.Logger("syncer")

var (
	ErrNotAskedByMe   = errors.New("Error Get Sync Resp but not asked by me")
	ErrNoTaskWait     = errors.New("Error No Task Waiting Result")
	ErrNotAccept      = errors.New("Error The Result had been rejected")
	ErrIgnore         = errors.New("Ignore")
	ErrEpochMismatch  = errors.New("Error Epoch mismatch with what syncer expected")
	ErrConsusMismatch = errors.New("Error consensus session mismatch")
	ErrSyncerStatus   = errors.New("Error Get GetEpoch response but syncer status mismatch")
)

const TASK_TIMEOUT = 4 //seconds

type SyncerAction uint

const (
	ContinueGetEpoch SyncerAction = iota
	SyncDone
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
	TaskId     string
	Type       TaskType
	RetryCount uint
	Meta       interface{}
}

type SyncResult struct {
	TaskId     string
	Type       TaskType
	nextAction SyncerAction
}

type Gsyncer struct {
	GroupId string
	Status  int8

	CurrentTask *SyncTask

	//chan signals
	taskq      chan *SyncTask
	resultq    chan *SyncResult
	taskdone   chan struct{}
	stopnotify chan struct{}

	taskGenerators map[TaskType]func(args ...interface{}) (*SyncTask, error)
	tasksender     func(task *SyncTask) error //send task via network or others

	retrynext bool //workaround for rumexchange
}

func NewGsyncer(groupid string,
	taskGenerators map[TaskType]func(args ...interface{}) (*SyncTask, error),
	tasksender func(task *SyncTask) error) *Gsyncer {
	gsyncer_log.Debugf("<%s> NewGsyncer called", groupid)

	s := &Gsyncer{}
	s.GroupId = groupid
	s.Status = IDLE
	s.taskGenerators = taskGenerators
	s.tasksender = tasksender

	return s
}

func (s *Gsyncer) GetCurrentTask() (string, TaskType, uint, error) {
	if s.CurrentTask == nil {
		return "", 0, 0, ErrNoTaskWait
	}
	return s.CurrentTask.TaskId, s.CurrentTask.Type, s.CurrentTask.RetryCount, nil
}

func (s *Gsyncer) SetRetryWithNext(retrynext bool) {
	s.retrynext = retrynext
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
			ctx, cancel := context.WithTimeout(context.Background(), 2*TASK_TIMEOUT*time.Second)
			defer cancel()
			s.processTask(ctx, task)
		}
		gsyncer_log.Debugf("here??")
		s.stopnotify <- struct{}{}
	}()

	//resultq
	go func() {
		for result := range s.resultq {
			s.handleResult(result)
		}
		s.stopnotify <- struct{}{}
	}()
}

func (s *Gsyncer) handleResult(result *SyncResult) {
	gsyncer_log.Debugf("<%s> handleResult called, taskId <%s>", s.GroupId, result.TaskId)

	switch result.nextAction {
	case SyncDone:
		gsyncer_log.Debugf("<%s> sync done, set to IDLE", s.GroupId)
		s.CurrentTask = nil
		s.Status = IDLE
	case ContinueGetEpoch:
		//add next sync task to taskq
		gsyncer_log.Debugf("<%s> ContinueGetEpoch", s.GroupId)
		nextTask, _ := s.taskGenerators[GetEpoch]()
		s.AddTask(nextTask)
	}

	//send taskdone signal
	s.taskdone <- struct{}{}
}

func (s *Gsyncer) processTask(ctx context.Context, task *SyncTask) error {
	//TODO: close this goroutine when the processTask func return. add some defer signal?
	gsyncer_log.Debugf("processTask called, taskId <%s>, retry <%d>", task.TaskId, task.RetryCount)
	go func() {
		s.CurrentTask = task //set current task
		gsyncer_log.Debugf("Set current task %s %d", s.CurrentTask.TaskId, s.CurrentTask.Type)
		switch task.Type {
		case ConsensusSync:
			s.Status = CONSENSUS_SYNC
		case GetEpoch:
			s.Status = SYNCING_FORWARD
		}
		s.tasksender(task)
	}()

	select {
	case <-s.taskdone:
		return nil
	case <-ctx.Done():
		if s.Status != CLOSE {
			//a workround, should cancel the ctx for current task
			if s.CurrentTask != nil {
				gsyncer_log.Debugf("task <%s> timeout,  retry now", task.TaskId)
				if task.Type == GetEpoch {
					//recreate timeout
					task.RetryCount++
					//put same task back to taskq again
					s.AddTask(task)
				} else if task.Type == ConsensusSync {
					//create a new consensus sync task
					newTask, _ := s.taskGenerators[ConsensusSync]()
					//keep the retry count
					newTask.RetryCount = task.RetryCount + 1
					s.AddTask(newTask)
				}

			}
		}
		return nil
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
