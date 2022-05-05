package conn

import (
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

func SendTrxWithoutRetry(groupId string, trx *quorumpb.Trx, channel PsConnChanel) (string, error) {
	connMgr, err := conn.GetConnMgr(groupId)
	if err != nil {
		return "", err
	}

	err = connMgr.SendTrxPubsub(trx, channel)
	if err != nil {
		return "", err
	}

	return trx.TrxId, nil
}
