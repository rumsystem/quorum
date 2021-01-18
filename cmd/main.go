package main

import (
    "os"
    "flag"
    "fmt"
	"context"
    "path/filepath"
    "strings"
	"crypto/rand"
	"github.com/spf13/viper"
	"github.com/golang/glog"
	maddr "github.com/multiformats/go-multiaddr"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

type addrList []maddr.Multiaddr
type Config struct {
	RendezvousString string
	BootstrapPeers   addrList
	ListenAddresses  string
	ProtocolID       string
    IsBootstrap     bool
}

type Keys struct{
	PrivKey p2pcrypto.PrivKey
	PubKey p2pcrypto.PubKey
}


func ParseFlags() (Config,error) {
    config := Config{ProtocolID:"/quorum/1.0.0"}
	flag.StringVar(&config.RendezvousString, "rendezvous", "some unique string",
		"Unique string to identify group of nodes. Share this with your friends to let them connect with you")
	flag.Var(&config.BootstrapPeers, "peer", "Adds a peer multiaddress to the bootstrap list")
	flag.StringVar(&config.ListenAddresses, "listen", "/ip4/127.0.0.1/tcp/4215", "Adds a multiaddress to the listen list")
    flag.BoolVar(&config.IsBootstrap, "bootstrap", false, "run a bootstrap node")
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


// isolates the complex initialization steps
//func constructPeerHost(ctx context.Context, id peer.ID, ps peerstore.Peerstore, options ...libp2p.Option) (host.Host, error) {
//	pkey := ps.PrivKey(id)
//	if pkey == nil {
//		return nil, fmt.Errorf("missing private key for node ID: %s", id.Pretty())
//	}
//	options = append([]libp2p.Option{libp2p.Identity(pkey), libp2p.Peerstore(ps)}, options...)
//	return libp2p.New(ctx, options...)
//}

func writekeysToconfig(priv p2pcrypto.PrivKey, pub p2pcrypto.PubKey) error{

	privkeybytes,err := p2pcrypto.MarshalPrivateKey(priv)
    if err != nil {
        return err
    }
	pubkeybytes,err := p2pcrypto.MarshalPublicKey(pub)
    if err != nil {
        return err
    }
    viper.Set("priv",p2pcrypto.ConfigEncodeKey(privkeybytes))
    viper.Set("pub",p2pcrypto.ConfigEncodeKey(pubkeybytes))
    viper.SafeWriteConfig()
    return nil
}

func loadKeys() (*Keys,error){
	viper.AddConfigPath(filepath.Dir("./config/"))
	viper.SetConfigName("keys")
	viper.SetConfigType("toml")
    err := viper.ReadInConfig()
    if err != nil {

	    glog.Infof("Keys files not found, generating new keypair..")
	    priv, pub, err := p2pcrypto.GenerateKeyPairWithReader(p2pcrypto.RSA, 4096, rand.Reader)
        if err != nil{
            return nil, err
        }
        writekeysToconfig(priv, pub)
    }
    err = viper.ReadInConfig()
    if err !=nil {
        return nil, err
    }

    privstr := viper.GetString("priv")
    pubstr := viper.GetString("pub")
	glog.Infof("Load keys from config")

    serializedpub,_ := p2pcrypto.ConfigDecodeKey(pubstr)
    pubfromconfig, err := p2pcrypto.UnmarshalPublicKey(serializedpub)
    if err !=nil {
        return nil, err
    }

    serializedpriv,_ := p2pcrypto.ConfigDecodeKey(privstr)
    privfromconfig, err := p2pcrypto.UnmarshalPrivateKey(serializedpriv)
    if err !=nil {
        return nil, err
    }
	return &Keys{PrivKey: privfromconfig, PubKey: pubfromconfig}, nil
}

func mainRet(config Config) int {
	ctx := context.Background()
	fmt.Println(ctx)
	keys,_ := loadKeys()
	fmt.Println(keys)
	privkeybytes,_ := p2pcrypto.MarshalPrivateKey(keys.PrivKey)
	fmt.Println(p2pcrypto.ConfigEncodeKey(privkeybytes))
    //https://github.com/ipfs/go-ipfs/blob/78c6dba9cc584c5f94d3c610ee95b57272df891f/cmd/ipfs/daemon.go#L360
    //node, err := core.NewNode(req.Context, ncfg)
    //https://github.com/ipfs/go-ipfs/blob/8e6358a4fac40577950260d0c7a7a5d57f4e90a9/core/builder.go#L27
    //ipfs: use fx to build an IPFS node https://github.com/uber-go/fx 
    //node.IPFS(ctx, cfg): https://github.com/ipfs/go-ipfs/blob/7588a6a52a789fa951e1c4916cee5c7a304912c2/core/node/groups.go#L307

    return 0
}

func main() {
	help := flag.Bool("h", false, "Display Help")
	version := flag.Bool("version", false, "Show the version")
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

	if *version {
        fmt.Println("1.0.0")
        return
    }
	os.Exit(mainRet(config))
}
