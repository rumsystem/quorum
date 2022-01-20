package chain

import (
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type User interface {
	Init(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface)
	UpdAnnounce(item *quorumpb.AnnounceItem) (string, error)
	UpdSchema(item *quorumpb.SchemaItem) (string, error)
	UpdProducer(item *quorumpb.ProducerItem) (string, error)
	UpdUser(item *quorumpb.UserItem) (string, error)
	UpdAppConfig(item *quorumpb.AppConfigItem) (string, error)
	PostToGroup(content proto.Message, encryptto ...[]string) (string, error)
	AddBlock(block *quorumpb.Block) error
	UpdChainConfig(item *quorumpb.ChainConfigItem) (string, error)
}
