package pubsubconn

type PubSubConn interface {
	JoinChannel(cId string, psConnCIface ChainDataHandlerPsconnIface) error
	Publish(data []byte) error
}
