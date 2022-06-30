package def

import (
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type TrxFactoryIface interface {
	GetBlockProducedTrx(keyalias string, blk *quorumpb.Block) (*quorumpb.Trx, error)
	GetAnnounceTrx(keyalias string, item *quorumpb.AnnounceItem) (*quorumpb.Trx, error)
	GetChainConfigTrx(keyalias string, item *quorumpb.ChainConfigItem) (*quorumpb.Trx, error)
	GetUpdSchemaTrx(keyalias string, item *quorumpb.SchemaItem) (*quorumpb.Trx, error)
	GetRegProducerTrx(keyalias string, item *quorumpb.ProducerItem) (*quorumpb.Trx, error)
	GetUpdAppConfigTrx(keyalias string, item *quorumpb.AppConfigItem) (*quorumpb.Trx, error)
	GetRegUserTrx(keyalias string, item *quorumpb.UserItem) (*quorumpb.Trx, error)
	GetPostAnyTrx(keyalias string, content proto.Message, encryptto ...[]string) (*quorumpb.Trx, error)
	GetReqBlockForwardTrx(keyalias string, block *quorumpb.Block) (*quorumpb.Trx, error)
	GetReqBlockBackwardTrx(keyalias string, block *quorumpb.Block) (*quorumpb.Trx, error)
}
