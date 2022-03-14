package chain

import (
	"time"

	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

type PublishQueueItem struct {
	groupId string
	state   string

	trx *quorumpb.Trx
}

var PublishQueueItemStatePending = "PENDING"
var PublishQueueItemStateSuccess = "SUCCESS"
var PublishQueueItemStateFail = "FAIL"

func InitPublishQueueWatcher(done chan bool) {
	// hard coded to 10s
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				doRefresh()
			}
		}
	}()
}

func doRefresh() {
	// TODO:
}

func TrxEnqueue() {
	// TODO: store/update the trx in db after published
}

func TrxDequeue() {
	// TODO: if success, remove from db
	// otherwise, retry with some backoff strategy
	// retry means: republish and enqueue again
}
