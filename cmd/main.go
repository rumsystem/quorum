package main

import (
    "os"
    "flag"
    "fmt"
    "strings"
	maddr "github.com/multiformats/go-multiaddr"
)

type addrList []maddr.Multiaddr
type Config struct {
	RendezvousString string
	BootstrapPeers   addrList
	ListenAddresses  string
	ProtocolID       string
}


func ParseFlags() (Config,error) {
    config := Config{ProtocolID:"/quorum/1.0.0"}
	flag.StringVar(&config.RendezvousString, "rendezvous", "some unique string",
		"Unique string to identify group of nodes. Share this with your friends to let them connect with you")
	flag.Var(&config.BootstrapPeers, "peer", "Adds a peer multiaddress to the bootstrap list")
	flag.StringVar(&config.ListenAddresses, "listen", "/ip4/127.0.0.1/tcp/4215", "Adds a multiaddress to the listen list")
	flag.Parse()

	return config, nil
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


func mainRet() int {
	//rand.Seed(time.Now().UnixNano())
    return 0
}

func main() {
	help := flag.Bool("h", false, "Display Help")
	config, err := ParseFlags()
	if err != nil {
		panic(err)
	}
	if *help {
		fmt.Println("Output a help ")
		fmt.Println()
		fmt.Println("Usage:...")
		flag.PrintDefaults()
		return
	}
    fmt.Println(config)
	os.Exit(mainRet())
}
