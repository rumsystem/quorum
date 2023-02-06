package chain

import (
	"context"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/logging"
)

var gsyncer_log = logging.Logger("syncer")

type SyncTask struct {
	SessionId  string
	TaskId     string
	RetryCount uint
	Meta       interface{}
}

type SyncMsg struct {
	TaskId string
	Data   interface{}
}

type SyncerStatus uint

const (
	IDLE SyncerStatus = iota
	SYNCING
	CLOSED
)

type GSyncer struct {
	GroupId   string
	SessionId string
	Status    SyncerStatus

	CurrentTask *SyncTask
	taskTimeout int

	//chan signals
	taskq      chan *SyncTask
	msgq       chan *SyncMsg
	taskdone   chan struct{}
	stopnotify chan struct{}

	taskGenerator func(args ...interface{}) *SyncTask
	taskSender    func(task *SyncTask) error
	msgHandler    func(msg *SyncMsg, syncer *GSyncer) error
}

func NewGsyncer(groupid string,
	sessionId string,
	taskGenerator func(args ...interface{}) *SyncTask,
	taskSender func(task *SyncTask) error,
	msgHandler func(msg *SyncMsg, syncer *GSyncer) error,
	taskTimeout int) *GSyncer {
	gsyncer_log.Debugf("<%s> NewGSyncer called", groupid)

	s := &GSyncer{}
	s.GroupId = groupid
	s.SessionId = sessionId
	s.Status = IDLE
	s.CurrentTask = nil
	s.taskTimeout = taskTimeout
	s.taskGenerator = taskGenerator
	s.taskSender = taskSender
	s.msgHandler = msgHandler

	gsyncer_log.Debugf("<%s> Init gsyncer channels", s.GroupId)
	s.taskq = make(chan *SyncTask)
	s.msgq = make(chan *SyncMsg, 3)
	s.taskdone = make(chan struct{})
	s.stopnotify = make(chan struct{})

	//taskq
	go func() {
		for task := range s.taskq {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.taskTimeout)*time.Second)
			defer cancel()
			s.runTask(ctx, task)
		}
		s.stopnotify <- struct{}{}
	}()

	//resultq
	go func() {
		for msg := range s.msgq {
			s.handleMsg(msg)
		}
		s.stopnotify <- struct{}{}
	}()

	return s
}

func (s *GSyncer) GetCurrentTask() SyncTask {
	return *s.CurrentTask
}

func (s *GSyncer) Next() {
	gsyncer_log.Debugf("<%s> Next called, gsyncer session <%s>", s.GroupId, s.SessionId)
	nextTask := s.taskGenerator(s.SessionId)
	s.AddTask(nextTask)
}

func (s *GSyncer) Stop() {
	gsyncer_log.Debugf("<%s> Stop called", s.GroupId)
	s.Status = CLOSED
	safeCloseTaskQ(s.taskq)
	safeCloseMsgQ(s.msgq)
	safeClose(s.taskdone)
	if s.stopnotify != nil {
		signcount := 0
		for _ = range s.stopnotify {
			signcount++
			//wait stop sign and set idle
			if signcount == 2 { // taskq and resultq stopped
				close(s.stopnotify)
				gsyncer_log.Debugf("gsyncer <%s> stop success.", s.GroupId)
			}
		}
	}
}

func (s *GSyncer) CurrentTaskDone() {
	s.taskdone <- struct{}{}
	s.CurrentTask = nil
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

func safeCloseTaskQ(ch chan *SyncTask) (recovered bool) {
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

func safeCloseMsgQ(ch chan *SyncMsg) (recovered bool) {
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

func (s *GSyncer) runTask(ctx context.Context, task *SyncTask) error {
	//TODO: close this goroutine when the processTask func return. add some defer signal?
	gsyncer_log.Debugf("runTask called, taskId <%s>, retry <%d>", task.TaskId, task.RetryCount)
	go func() {
		s.CurrentTask = task //set current task
		s.taskSender(task)
	}()

	select {
	case <-s.taskdone:
		return nil
	case <-ctx.Done():
		if s.Status != CLOSED {
			//a workround, should cancel the ctx for current task
			if s.CurrentTask != nil {
				gsyncer_log.Debugf("task <%s> timeout,  retry now", task.TaskId)
				s.AddTask(task)
			}
		}
		return nil
	}
}

func (s *GSyncer) handleMsg(msg *SyncMsg) {
	gsyncer_log.Debugf("<%s> handleMsg called, taskId <%s>", s.GroupId, msg.TaskId)
	s.msgHandler(msg, s)

	/*
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
	*/
}

func (s *GSyncer) AddTask(task *SyncTask) {
	gsyncer_log.Debugf("Gsyncer addTask called")
	go func() {
		if s.Status != CLOSED {
			s.taskq <- task
		}
	}()
}

func (s *GSyncer) AddMsg(msg *SyncMsg) {
	go func() {
		if s.Status != CLOSED {
			s.msgq <- msg
		}
	}()
}
