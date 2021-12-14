package handlers

import (
	"context"
	"fmt"
	"github.com/rumsystem/quorum/internal/pkg/p2p"
)

func RexTest(node *p2p.Node) ([]string, error) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := node.ConnectRex(ctx, 3)

	fmt.Println(err)
	return []string{"ok"}, nil
}
