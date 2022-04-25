package handlers

import (
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
)

type PubQueueInfo struct {
	GroupId string
	Data    []*chain.PublishQueueItem
}

func GetPubQueue(groupId string, status string, trxId string) (*PubQueueInfo, error) {
	items, err := chain.GetPubQueueWatcher().GetGroupItems(groupId, status, trxId)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		item.StorageType = item.Trx.StorageType.String()
	}

	ret := PubQueueInfo{groupId, items}

	return &ret, nil
}

func PubQueueAck(trxIds []string) ([]string, error) {
	return chain.GetPubQueueWatcher().Ack(trxIds)
}
