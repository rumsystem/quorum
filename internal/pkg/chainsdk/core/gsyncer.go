package chain

import (
	"context"
	"errors"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/logging"
)

var gsyncer_log = logging.Logger("syncer")

var (
	ErrSyncDone = errors.New("Error Signal Sync Done")
)

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
	s.taskq = make(chan *SyncTask)
	s.resultq = make(chan *SyncResult, 3)
	return s
}
func (s *Gsyncer) Stop() {
	close(s.taskq)
	close(s.resultq)
}

func (s *Gsyncer) Start() {
	gsyncer_log.Debugf("<%s> Gsyncer Start", s.GroupId)
	go func() {
		for task := range s.taskq {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			err := s.processTask(ctx, task)
			if err == nil {
				gsyncer_log.Debugf("<%s> process task done %s", s.GroupId, task.Id)
				//fake result send to the result queue
				//data := BlockSyncResult{BlockId: fmt.Sprintf("test_block_id_%s", task.Id)}
				//sr := &SyncResult{Id: task.Id, TaskId: task.Id, Data: data}
				//s.AddResult(sr)
				//will be replaced by real task result
			} else {
				//test try to retry this task
				s.AddTask(task)
				gsyncer_log.Errorf("<%s> task process %s error: %s", s.GroupId, task.Id, err)
			}
		}
	}()

	go func() {
		for result := range s.resultq {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			nexttaskid, err := s.processResult(ctx, result)
			//TODO: err with STOP signal, set to IDLE and pause
			if err == nil {
				//test try to add next task
				gsyncer_log.Debugf("<%s> process result done %s", s.GroupId, result.Id)
				if nexttaskid == "" {
					gsyncer_log.Errorf("nextTask can't be null, skip")
					continue
				}

				nexttask, err := s.nextTask(nexttaskid)
				if err != nil {
					gsyncer_log.Errorf("nextTask error:%s", err)
					continue
				}
				s.AddTask(nexttask)
			} else if err == ErrSyncDone {
				gsyncer_log.Infof("<%s> result %s is Sync Pause Signal", s.GroupId, result.Id)
				//SyncPause, stop add next task, pause
			} else {
				nexttask, terr := s.nextTask("") //the taskid shoule be inclued in the result, which need to upgrade all publicnode. so a workaround, pass a "" to get a retry task. (runner will try to maintain a taskid)
				if terr != nil {
					gsyncer_log.Errorf("nextTask error:%s", terr)
					continue
				}
				//test try to retry this task
				s.AddTask(nexttask)
				gsyncer_log.Errorf("<%s> result process %s error: %s", s.GroupId, result.Id, err)
			}
		}
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
			gsyncer_log.Warnf("<%s> done result: %s", s.GroupId, result.Id)
		default:
			return
		}
	}()

	select {
	case <-resultdone:
		return nexttaskid, err
	case <-ctx.Done():
		return "", errors.New("Result Timeout")
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
			gsyncer_log.Debugf("<%s> done %s task", s.GroupId, task.Id)
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
