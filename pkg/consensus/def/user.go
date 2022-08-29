package def

import (
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

type User interface {
	NewUser(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface)
	AddBlock(block *quorumpb.Block) error
}
