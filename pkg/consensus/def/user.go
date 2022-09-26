package def

import (
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type User interface {
	NewUser(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface)
	AddBlock(block *quorumpb.Block) error
}
