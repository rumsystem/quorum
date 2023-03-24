package handlers

import (
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	"github.com/rumsystem/quorum/pkg/pb"
)

type GetTrxParam struct {
	GroupId string `param:"group_id" validate:"required,uuid4" example:"ac0eea7c-2f3c-4c67-80b3-136e46b924a8"`
	TrxId   string `param:"trx_id" validate:"required,uuid4" example:"22d5c38d-5921-4b75-8562-c110dcfd5ee8"`
}

func GetTrx(groupid string, trxid string) (*pb.Trx, []int64, error) {
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
