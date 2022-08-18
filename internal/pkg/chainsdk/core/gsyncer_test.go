package chain

import (
	"fmt"
	"testing"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/logging"
)

func TestInit(t *testing.T) {
	logging.SetLogLevel("gsyncer", "debug")
	gs := NewGsyncer("3bb7a3be-d145-44af-94cf-e64b992ff8f0") //test groupid
	gs.Start()
	for i := 0; i < 500; i++ {
		taskmeta := BlockSyncTask{BlockId: fmt.Sprintf("00000000-0000-0000-0000-000000000001_%d", i), Direction: Next}
		go gs.AddTask(&SyncTask{Meta: taskmeta, Id: fmt.Sprintf("%d", i)})
		time.Sleep(1 * time.Second)
	}
	select {}
}
