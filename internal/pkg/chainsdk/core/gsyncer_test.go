package chain

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/logging"
)

var MaxTaskId int = 12

var taskresultcache map[string]*SyncResult
var gs *Gsyncer

//define how to get next task, for example, taskid+1
func GetNextTask(taskid string) (*SyncTask, error) {
	nextid, _ := strconv.Atoi(taskid)
	if nextid >= MaxTaskId {
		return nil, errors.New("reach the max task id")
	}
	nextid++
	taskmeta := BlockSyncTask{BlockId: fmt.Sprintf("00000000-0000-0000-0000-000000000001_%d", nextid), Direction: Next}
	return &SyncTask{Meta: taskmeta, Id: fmt.Sprintf("%d", nextid)}, nil
}

func ResultReceiver(result *SyncResult) (string, error) {
	taskresultcache[result.Id] = result
	return result.Id, nil
}

func TaskSender(task *SyncTask) error {
	result := &SyncResult{Id: task.Id}
	gs.AddResult(result)
	return nil
}

func TestTaskResult(t *testing.T) {
	taskresultcache = make(map[string]*SyncResult)
	logging.SetLogLevel("gsyncer", "debug")

	gs = NewGsyncer("3bb7a3be-d145-44af-94cf-e64b992ff8f0", GetNextTask, ResultReceiver, TaskSender) //test groupid
	gs.Start()
	i := 0
	taskmeta := BlockSyncTask{BlockId: fmt.Sprintf("00000000-0000-0000-0000-000000000001_%d", i), Direction: Next}
	gs.AddTask(&SyncTask{Meta: taskmeta, Id: fmt.Sprintf("%d", i)})
	for {
		time.Sleep(2 * time.Second)
		if len(taskresultcache) >= MaxTaskId { //success
			gsyncer_log.Info("all %d result received", MaxTaskId)
			break

		}
	}
	gs.Stop() //cleanup
}
