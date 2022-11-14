package handlers

import (
	"errors"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/pkg/pb"
)

func GetTrx(groupid string, trxid string) (*pb.Trx, []int64, error) {
	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		trx, nonces, err := group.GetTrx(trxid)
		if err != nil || trx != nil {

			return trx, nonces, err
		}
		return group.GetTrxFromCache(trxid)

	} else {
		return nil, nil, errors.New(fmt.Sprintf("Group %s not exist", groupid))
	}
}
