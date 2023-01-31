package cli

import (
	"strings"

	maddr "github.com/multiformats/go-multiaddr"
)

type AddrList []maddr.Multiaddr

type FullNodeFlag struct {
	RendezvousString string
	BootstrapPeers   AddrList
	ListenAddresses  AddrList
	APIHost          string
	APIPort          uint
	CertDir          string
	ZeroAccessKey    string
	ProtocolID       string
	PeerName         string
	JsonTracer       string
	IsDebug          bool
	ConfigDir        string
	DataDir          string
	KeyStoreDir      string
	KeyStoreName     string
	KeyStorePwd      string
	AutoAck          bool
	EnableRelay      bool

	IsConsensusTest bool
}

// TBD remove unused flags
type BootstrapNodeFlag struct {
	RendezvousString string
	BootstrapPeers   AddrList
	ListenAddresses  AddrList
	APIHost          string
	APIPort          uint
	CertDir          string
	ZeroAccessKey    string
	ProtocolID       string
	PeerName         string
	JsonTracer       string
	IsDebug          bool
	ConfigDir        string
	DataDir          string
	KeyStoreDir      string
	KeyStoreName     string
	KeyStorePwd      string
	AutoAck          bool
	EnableRelay      bool
}

type LightnodeFlag struct {
	PeerName     string
	ConfigDir    string
	DataDir      string
	KeyStoreDir  string
	KeyStoreName string
	KeyStorePwd  string
	APIHost      string
	APIPort      uint
	JsonTracer   string
	IsDebug      bool
}

type RelayNodeFlag struct {
	BootstrapPeers  AddrList
	ListenAddresses AddrList
	APIHost         string
	APIPort         uint
	PeerName        string
	ConfigDir       string
	DataDir         string
	KeyStoreDir     string
	KeyStoreName    string
	KeyStorePwd     string
	IsDebug         bool
}

type ProducerNodeFlag struct {
	RendezvousString string
	BootstrapPeers   AddrList
	ListenAddresses  AddrList
	APIHost          string
	APIPort          uint
	CertDir          string
	ZeroAccessKey    string
	ProtocolID       string
	PeerName         string
	JsonTracer       string
	IsDebug          bool
	ConfigDir        string
	DataDir          string
	KeyStoreDir      string
	KeyStoreName     string
	KeyStorePwd      string

	IsConsensusTest bool
}

func (al *AddrList) String() string {
	strs := make([]string, len(*al))
	for i, addr := range *al {
		strs[i] = addr.String()
	}
	return strings.Join(strs, ",")
}

func (al *AddrList) Set(value string) error {
	addrlist := strings.Split(value, ",")

	for _, v := range addrlist {
		addr, err := maddr.NewMultiaddr(v)
		if err != nil {
			return err
		}
		*al = append(*al, addr)
	}
	return nil
}

func (al *AddrList) Type() string {
	return "AddrList"
}
