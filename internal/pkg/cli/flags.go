package cli

import (
	"flag"
	"log"
	"path/filepath"
	"strings"

	maddr "github.com/multiformats/go-multiaddr"
)

type AddrList []maddr.Multiaddr

type Config struct {
	RendezvousString     string
	BootstrapPeers       AddrList
	ListenAddresses      AddrList
	APIHost              string
	APIPort              uint
	CertDir              string
	ZeroAccessKey        string
	NodeAPIListenAddress string
	ProtocolID           string
	IsBootstrap          bool
	IsRexTestMode        bool
	PeerName             string
	JsonTracer           string
	IsDebug              bool
	ConfigDir            string
	DataDir              string
	IsPing               bool
	KeyStoreDir          string
	KeyStoreName         string
	AutoAck              bool
	EnableRelay          bool
	EnableRelayService   bool
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

var quorumConfig Config

func GetConfig() Config {
	return quorumConfig
}

func ParseFlags() (Config, error) {
	config := Config{ProtocolID: "/quorum/1.0.0"}
	flag.StringVar(&config.RendezvousString, "rendezvous", "e6629921-b5cd-4855-9fcd-08bcc39caef7", //e6629921-b5cd-4855-9fcd-08bcc39caef7 default quorum rendezvous
		"Unique string to identify group of nodes. Share this with your friends to let them connect with you")
	flag.Var(&config.BootstrapPeers, "peer", "Adds a peer multiaddress to the bootstrap list")
	flag.Var(&config.ListenAddresses, "listen", "Adds a multiaddress to the listen list, e.g.: `-listen /ip4/127.0.0.1/tcp/4215 -listen /ip/127.0.0.1/tcp/5215/ws`")
	flag.StringVar(&config.APIHost, "apihost", "", "Domain or public ip addresses for api server")
	flag.UintVar(&config.APIPort, "apiport", 5215, "api server listen port")
	flag.StringVar(&config.CertDir, "certdir", "certs", "ssl certificate directory")
	flag.StringVar(&config.ZeroAccessKey, "zerosslaccesskey", "", "zerossl access key, get from: https://app.zerossl.com/developer")
	flag.StringVar(&config.NodeAPIListenAddress, "nodeapilisten", ":5216", "Adds a multiaddress to the listen list for nodesdk")
	flag.StringVar(&config.PeerName, "peername", "peer", "peername")
	flag.StringVar(&config.ConfigDir, "configdir", "./config/", "config and keys dir")
	flag.StringVar(&config.DataDir, "datadir", "./data/", "config dir")
	flag.StringVar(&config.KeyStoreDir, "keystoredir", "./keystore/", "keystore dir")
	flag.StringVar(&config.KeyStoreName, "keystorename", "defaultkeystore", "keystore name")
	flag.StringVar(&config.JsonTracer, "jsontracer", "", "output tracer data to a json file")
	flag.BoolVar(&config.IsBootstrap, "bootstrap", false, "run a bootstrap node")
	flag.BoolVar(&config.IsRexTestMode, "rextest", false, "RumExchange Test Mode")
	flag.BoolVar(&config.IsDebug, "debug", false, "show debug log")
	flag.BoolVar(&config.AutoAck, "autoack", false, "auto ack the transactions in pubqueue")
	flag.BoolVar(&config.EnableRelay, "autorelay", false, "enable relay")
	flag.BoolVar(&config.EnableRelayService, "relayservice", false, "provide relay service")

	flag.Parse()

	configDir, err := filepath.Abs(config.ConfigDir)
	if err != nil {
		log.Fatalf("get absolute path for config dir failed: %s", err)
	}
	config.ConfigDir = configDir

	dataDir, err := filepath.Abs(config.DataDir)
	if err != nil {
		log.Fatalf("get absolute path for data dir failed: %s", err)
	}
	config.DataDir = dataDir

	quorumConfig = config
	return config, nil
}

type RelayNodeConfig struct {
	BootstrapPeers  AddrList
	ListenAddresses AddrList
	PeerName        string
	IsDebug         bool
	ConfigDir       string
	KeyStoreDir     string
	KeyStoreName    string
}

func ParseRelayNodeFlags() (RelayNodeConfig, error) {
	config := RelayNodeConfig{}
	flag.Var(&config.BootstrapPeers, "peer", "Adds a peer multiaddress to the bootstrap list")
	flag.Var(&config.ListenAddresses, "listen", "Adds a multiaddress to the listen list, e.g.: `-listen /ip4/127.0.0.1/tcp/4215 -listen /ip/127.0.0.1/tcp/5215/ws`")
	flag.StringVar(&config.PeerName, "peername", "peer", "peername")
	flag.StringVar(&config.ConfigDir, "configdir", "./config/", "config and keys dir")
	flag.StringVar(&config.KeyStoreDir, "keystoredir", "./keystore/", "keystore dir")
	flag.StringVar(&config.KeyStoreName, "keystorename", "defaultkeystore", "keystore name")
	flag.BoolVar(&config.IsDebug, "debug", false, "show debug log")

	flag.Parse()

	configDir, err := filepath.Abs(config.ConfigDir)
	if err != nil {
		log.Fatalf("get absolute path for config dir failed: %s", err)
	}
	config.ConfigDir = configDir
	return config, nil
}
