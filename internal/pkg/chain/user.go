package chain

import (
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type User interface {
	Init(grp *Group)

	UpdAnnounce(item *quorumpb.AnnounceItem) (string, error)
	UpdBlkList(item *quorumpb.DenyUserItem) (string, error)
	UpdSchema(item *quorumpb.SchemaItem) (string, error)
	UpdProducer(item *quorumpb.ProducerItem) (string, error)
	PostToGroup(content proto.Message) (string, error)
	AddBlock(block *quorumpb.Block) error
}
