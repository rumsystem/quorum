package handlers

import (
	"context"
	"fmt"
	"github.com/rumsystem/quorum/internal/pkg/p2p"
)

func RexTest(node *p2p.Node) ([]string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if node.RumExchange != nil {
		//err := node.RumExchange.ConnectRex(ctx, 3)
		node.RumExchange.ConnectRex(ctx, 3)
		//fmt.Println(err)
		fmt.Println("call pingpoing")
		node.RumExchange.PingPong("16Uiu2HAm4U6Ymx5nNifPVBgn7ZaofXGmN9EEFtay7KjWtq64gZcM")
		return []string{"ok"}, nil
	} else {
		return []string{"not support rumexchange"}, nil
	}

}
