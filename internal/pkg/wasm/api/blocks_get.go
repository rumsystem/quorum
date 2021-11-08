//go:build js && wasm
// +build js,wasm

package api

import (
	"errors"
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/chain"
	"github.com/rumsystem/quorum/internal/pkg/pb"
)

func GetBlockById(gid string, bid string) (block *pb.Block, err error) {
	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[gid]; ok {
		block, err := group.GetBlock(bid)
		if err != nil {
			return nil, err
		}

		return block, nil
	} else {
		return nil, errors.New(fmt.Sprintf("Group %s not exist", gid))
	}
}
