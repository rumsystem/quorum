package chain

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/logging"
)

var gsyncer_log = logging.Logger("syncer")

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

type BlockSyncResult struct {
	BlockId string
}

type SyncResult struct {
	Data   interface{}
	Id     string
	TaskId string
}

type Gsyncer struct {
	nodeName       string
	GroupId        string
	Status         int8
	retryCount     int8
	taskq          chan *SyncTask
	resultq        chan *SyncResult
	nextTask       func(taskid string) (*SyncTask, error)
	resultreceiver func(result *SyncResult) error
	tasksender     func(task *SyncTask) error
}

func NewGsyncer(groupid string, getTask func(taskid string) (*SyncTask, error), resultreceiver func(result *SyncResult) error, tasksender func(task *SyncTask) error) *Gsyncer {
	gsyncer_log.Debugf("<%s> NewGsyncer called", groupid)
	s := &Gsyncer{}
	s.Status = IDLE
	s.GroupId = groupid
	s.nextTask = getTask
	s.resultreceiver = resultreceiver
	s.tasksender = tasksender
	s.retryCount = 0
	s.taskq = make(chan *SyncTask)
	s.resultq = make(chan *SyncResult, 3)
	return s
}
func (s *Gsyncer) Stop() {
	close(s.taskq)
	close(s.resultq)
}

func (s *Gsyncer) Start() {
	gsyncer_log.Infof("<%s> Gsyncer Start", s.GroupId)
	go func() {
		for task := range s.taskq {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			err := s.processTask(ctx, task)
			if err == nil {
				gsyncer_log.Infof("<%s> process task done %s", s.GroupId, task.Id)
				//fake result send to the result queue
				//data := BlockSyncResult{BlockId: fmt.Sprintf("test_block_id_%s", task.Id)}
				//sr := &SyncResult{Id: task.Id, TaskId: task.Id, Data: data}
				//s.AddResult(sr)
				//will be replaced by real task result
			} else {
				//test try to retry this task
				taskmeta := BlockSyncTask{BlockId: fmt.Sprintf("00000000-0000-0000-0000-000000000001_%s", task.Id), Direction: Next}
				s.AddTask(&SyncTask{Meta: taskmeta, Id: task.Id})
				gsyncer_log.Errorf("<%s> task process %s error: %s", s.GroupId, task.Id, err)
			}
		}
	}()

	go func() {
		for result := range s.resultq {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			err := s.processResult(ctx, result)
			if err == nil {
				//test try to add next task
				gsyncer_log.Infof("<%s> process result done %s", s.GroupId, result.Id)
				nexttask, err := s.nextTask(result.TaskId)
				if err != nil {
					gsyncer_log.Errorf("nextTask error:%s", err)
					continue
				}
				s.AddTask(nexttask)
			} else {
				//test try to retry this task
				taskmeta := BlockSyncTask{BlockId: fmt.Sprintf("00000000-0000-0000-0000-000000000001_%s", result.Id), Direction: Next}
				s.AddTask(&SyncTask{Meta: taskmeta, Id: result.TaskId})
				gsyncer_log.Errorf("<%s> result process %s error: %s", s.GroupId, result.Id, err)
			}
		}
	}()
}
func (s *Gsyncer) processResult(ctx context.Context, result *SyncResult) error {
	resultdone := make(chan struct{})
	var err error
	go func() {
		blocktaskresult, ok := result.Data.(BlockSyncResult)
		if ok == true {
			v := rand.Intn(2)
			time.Sleep(time.Duration(v) * time.Second) // fake workload
			err = s.resultreceiver(result)
			//try to save the result to db
		} else {
			gsyncer_log.Warnf("<%s> Unsupported result", result.Id)
		}
		select {
		case resultdone <- struct{}{}:
			gsyncer_log.Warnf("<%s> done %s result", s.GroupId, blocktaskresult.BlockId)
		default:
			return
		}
	}()

	select {
	case <-resultdone:
		return err
	case <-ctx.Done():
		return errors.New("Result Timeout")
	}

}

func (s *Gsyncer) processTask(ctx context.Context, task *SyncTask) error {
	taskdone := make(chan struct{})
	go func() {
		s.tasksender(task)
		//blocktask, ok := task.Meta.(BlockSyncTask)
		//if ok == true {
		//	//replace with  real workload
		//	//v := rand.Intn(5) + 1
		//	//time.Sleep(time.Duration(v) * time.Second) // fake workload
		//} else {
		//	fmt.Println("===task.Meta")
		//	fmt.Println(task.Meta)
		//	gsyncer_log.Warnf("<%s> Unsupported task", s.GroupId, task.Id)
		//}
		select {
		case taskdone <- struct{}{}:
			gsyncer_log.Warnf("<%s> done %s task", s.GroupId, task.Id)
		default:
			return
		}
	}()

	select {
	case <-taskdone:
		return nil
	case <-ctx.Done():
		return errors.New("Task Timeout")
	}
}

func (s *Gsyncer) AddTask(task *SyncTask) {
	go func() {
		s.taskq <- task
	}()
}

func (s *Gsyncer) AddResult(result *SyncResult) {
	s.resultq <- result
}
