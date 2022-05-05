package def

import (
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type TrxFactoryIface interface {
	GetBlockProducedTrx(blk *quorumpb.Block) (*quorumpb.Trx, error)
	GetAnnounceTrx(item *quorumpb.AnnounceItem) (*quorumpb.Trx, error)
	GetChainConfigTrx(item *quorumpb.ChainConfigItem) (*quorumpb.Trx, error)
	GetUpdSchemaTrx(item *quorumpb.SchemaItem) (*quorumpb.Trx, error)
	GetRegProducerTrx(item *quorumpb.ProducerItem) (*quorumpb.Trx, error)
	GetUpdAppConfigTrx(item *quorumpb.AppConfigItem) (*quorumpb.Trx, error)
	GetRegUserTrx(item *quorumpb.UserItem) (*quorumpb.Trx, error)
	GetPostAnyTrx(content proto.Message, encryptto ...[]string) (*quorumpb.Trx, error)
	GetReqBlockForwardTrx(block *quorumpb.Block) (*quorumpb.Trx, error)
	GetReqBlockBackwardTrx(block *quorumpb.Block) (*quorumpb.Trx, error)
}
