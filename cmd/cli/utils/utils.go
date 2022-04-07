package utils

import (
	"time"

	"github.com/rumsystem/quorum/cmd/cli/api"
	"github.com/rumsystem/quorum/internal/pkg/chain"
)

// this function will check trx in pubqueue
// once success, it will ack
// also, it will ack trx which failed > 10 times
func CheckTrx(groupId string, trxId string, status string) ([]string, error) {
	select {
	case <-time.After(30 * time.Second):
		qInfo, err := api.GetPubQueue(groupId, trxId, status)
		if err != nil {
			return nil, err
		}
		if len(qInfo.Data) == 0 {
			return nil, nil
		}
		successed := []string{}
		rejected := []string{}
		for _, item := range qInfo.Data {
			if item.State == chain.PublishQueueItemStateSuccess {
				successed = append(successed, item.Trx.TrxId)
			}
			if item.RetryCount > chain.MAX_RETRY_COUNT && item.State == chain.PublishQueueItemStateFail {
				rejected = append(rejected, item.Trx.TrxId)
			}
		}

		acked := append(successed, rejected...)
		if len(acked) > 0 {
			_, err = api.PubQueueAck(acked)
			if err != nil {
				return nil, err
			}
			return acked, nil
		}
		return nil, nil
	}
}
