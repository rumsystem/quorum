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
	SkipPeers        string
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
}

func (al *AddrList) String() string {
	strs := make([]string, len(*al))
	for i, addr := range *al {
		strs[i] = addr.String()
	}
	return strings.Join(strs, ",")
}

func (al *AddrList) Set(value string) error {
	tmpAL, err := ParseAddrList(value)
	if err != nil {
		return err
	}

	*al = *tmpAL
	return nil
}

func (al *AddrList) Type() string {
	return "AddrList"
}

func ParseAddrList(s string) (*AddrList, error) {
	addrlist := strings.Split(s, ",")
	var al AddrList

	for _, v := range addrlist {
		addr, err := maddr.NewMultiaddr(v)
		if err != nil {
			return nil, err
		}
		al = append(al, addr)
	}

	return &al, nil
}
