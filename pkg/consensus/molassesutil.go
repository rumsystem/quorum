package consensus

import (
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

//update resend count (+1) for all trxs
func UpdateResendCount(trxs []*quorumpb.Trx) ([]*quorumpb.Trx, error) {
	for _, trx := range trxs {
		trx.ResendCount++
	}
	return trxs, nil
}
