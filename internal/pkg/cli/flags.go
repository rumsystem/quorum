package cli

import (
    "flag"
    "strings"
	maddr "github.com/multiformats/go-multiaddr"
)

type addrList []maddr.Multiaddr

type Config struct {
	RendezvousString string
	BootstrapPeers   addrList
	ListenAddresses  string
	ProtocolID       string
    IsBootstrap     bool
    PeerName        string
}

func (al *addrList) String() string {
	strs := make([]string, len(*al))
	for i, addr := range *al {
		strs[i] = addr.String()
	}
	return strings.Join(strs, ",")
}

func (al *addrList) Set(value string) error {
	addr, err := maddr.NewMultiaddr(value)
	if err != nil {
		return err
	}
	*al = append(*al, addr)
	return nil
}


func ParseFlags() (Config,error) {
    config := Config{ProtocolID:"/quorum/1.0.0"}
	flag.StringVar(&config.RendezvousString, "rendezvous", "some unique string",
		"Unique string to identify group of nodes. Share this with your friends to let them connect with you")
	flag.Var(&config.BootstrapPeers, "peer", "Adds a peer multiaddress to the bootstrap list")
	flag.StringVar(&config.ListenAddresses, "listen", "/ip4/127.0.0.1/tcp/4215", "Adds a multiaddress to the listen list")
	flag.StringVar(&config.PeerName, "peername", "peer", "peername")
    flag.BoolVar(&config.IsBootstrap, "bootstrap", false, "run a bootstrap node")
	flag.Parse()

	return config, nil
}
