package pubsubconn

import chaindef "github.com/rumsystem/quorum/internal/pkg/chainsdk/def"

type PubSubConn interface {
	JoinChannel(cId string, cdhIface chaindef.ChainDataSyncIface) error
	Publish(data []byte) error
}
