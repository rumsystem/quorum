package handlers

import (
	"context"
	"fmt"

	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	"github.com/rumsystem/quorum/internal/pkg/p2p"
)

func RexInitSession(node *p2p.Node, groupId string, peerId string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if node.RumExchange != nil {
		fmt.Println("call rex init session")
		err := node.RumExchange.ConnectRex(ctx)
		if err != nil {
			return err
		}
		node.RumSession.InitSession(peerId, "prod_channel_"+groupId)
		groupmgr := chain.GetGroupMgr()
		group, ok := groupmgr.Groups[groupId]
		if ok == true {
			group.AskPeerId()
		}

		return nil
	} else {
		return fmt.Errorf("not support rumexchange")
	}

}
