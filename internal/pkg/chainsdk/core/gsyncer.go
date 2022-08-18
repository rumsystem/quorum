package chain

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/logging"
)

var gsyncer_log = logging.Logger("gsyncer")

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

type Gsyncer struct {
	nodeName   string
	GroupId    string
	Status     int8
	retryCount int8
	taskwg     sync.WaitGroup
	taskq      chan *SyncTask
	//syncNetworkType     conn.P2pNetworkType
}

func NewGsyncer(groupid string) *Gsyncer {
	gsyncer_log.Debugf("<%s> NewGsyncer called", groupid)
	s := &Gsyncer{}
	s.Status = IDLE
	s.GroupId = groupid
	s.retryCount = 0
	s.taskq = make(chan *SyncTask)
	return s
}

func (s *Gsyncer) Start() {
	gsyncer_log.Infof("<%s> Gsyncer Start", s.GroupId)
	go func() {
		defer close(s.taskq)
		for task := range s.taskq {
			s.taskwg.Add(1)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

			blocktask, ok := task.Meta.(BlockSyncTask)
			if ok == true {
				go s.doBlockTask(ctx, &blocktask)
				s.taskwg.Wait()
			} else {
				gsyncer_log.Warnf("<%s> Unsupported task", s.GroupId)
			}
			cancel()
		}
	}()
	gsyncer_log.Infof("<%s> Gsyncer end", s.GroupId)

}

func (s *Gsyncer) processBlockTask(ctx context.Context, task *BlockSyncTask, fn func(r string)) {
loop:
	select {
	case <-ctx.Done():
		{
			gsyncer_log.Infof("<%s> processTask %s end", task.BlockId)
			gsyncer_log.Infof("<%s> processTask break loop", task.BlockId)

			break loop
		}
	default:
		{
			v := rand.Intn(10) + 1
			gsyncer_log.Infof("test...sleep: %ds", v)
			time.Sleep(time.Duration(v) * time.Second)
			fn("done " + task.BlockId)
			gsyncer_log.Infof("process blocktask done %s ", task.BlockId)
		}
	}
	gsyncer_log.Infof("END: process blocktask exit %s ", task.BlockId)

}

func (s *Gsyncer) doBlockTask(ctx context.Context, task *BlockSyncTask) {
	ctx, cancel := context.WithCancel(ctx)
	go s.processBlockTask(ctx, task, func(result string) {
		fmt.Println("********received result ", result)
	})
	select {
	case <-ctx.Done():
		{
			gsyncer_log.Infof("blockTask %s timeout", task.BlockId)
			cancel()
		}

	}
	gsyncer_log.Infof("blockTask %s end", task.BlockId)
	defer s.taskwg.Done()
}

func (s *Gsyncer) AddTask(task *SyncTask) {
	s.taskq <- task
}
