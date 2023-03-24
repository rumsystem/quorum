package handlers

import (
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/pkg/pb"
)

func GetTrx(groupid string, trxid string) (*pb.Trx, error) {
	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		trx, err := group.GetTrx(trxid)
		if err != nil || trx != nil {
			return trx, err
		}
		return group.GetTrxFromCache(trxid)

	} else {
		return nil, fmt.Errorf("group %s not exist", groupid)
	}
}
