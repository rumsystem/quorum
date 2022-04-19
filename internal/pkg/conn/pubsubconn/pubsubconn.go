package pubsubconn

import iface "github.com/rumsystem/quorum/internal/pkg/chainsdk/chaindataciface"

type PubSubConn interface {
	JoinChannel(cId string, cdhIface iface.ChainDataHandlerIface) error
	Publish(data []byte) error
}
