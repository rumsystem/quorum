package handlers

import (
	"errors"
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/chain"
	"github.com/rumsystem/quorum/internal/pkg/pb"
)

func GetTrx(groupid string, trxid string) (*pb.Trx, error) {
	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[groupid]; ok {
		return group.GetTrx(trxid)
	} else {
		return nil, errors.New(fmt.Sprintf("Group %s not exist", groupid))
	}
}
