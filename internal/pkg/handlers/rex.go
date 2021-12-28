package handlers

import (
	"context"
	"fmt"
	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	"github.com/rumsystem/quorum/internal/pkg/p2p"
)

func RexTest(node *p2p.Node) ([]string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if node.RumExchange != nil {

		fmt.Println("call rextest")
		err := node.RumExchange.ConnectRex(ctx, 3)
		if err != nil {
			return []string{"err", err.Error()}, nil
		}
		peerid := "16Uiu2HAm4U6Ymx5nNifPVBgn7ZaofXGmN9EEFtay7KjWtq64gZcM"
		channelid := "my_private_channel"
		node.RumExchange.InitSession(peerid, channelid)
		ch := make(chan int)
		select {
		case <-ch:
			break

		}

		//node.RumExchange.PingPong()
		return []string{"ok"}, nil
	} else {
		return []string{"not support rumexchange"}, nil
	}

}

func RexInitSession(node *p2p.Node, groupId string, peerId string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if node.RumExchange != nil {
		fmt.Println("call rex init session")
		err := node.RumExchange.ConnectRex(ctx, 3)
		if err != nil {
			return err
		}
		node.RumExchange.InitSession(peerId, "prod_channel_"+groupId)
		groupmgr := chain.GetGroupMgr()
		group, ok := groupmgr.Groups[groupId]
		if ok == true {
			group.ChainCtx.AskPeerId()
		}

		return nil
	} else {
		return fmt.Errorf("not support rumexchange")
	}

}
