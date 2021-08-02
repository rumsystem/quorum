package testnode

import (
	"context"
	"fmt"
	"go/build"
	"io/ioutil"
	"log"
	"os"

	//"syscall"
	"time"

	"github.com/huo-ju/quorum/internal/pkg/cli"
	localcrypto "github.com/huo-ju/quorum/internal/pkg/crypto"
	"github.com/huo-ju/quorum/internal/pkg/options"
	"github.com/huo-ju/quorum/internal/pkg/p2p"
	"github.com/huo-ju/quorum/internal/pkg/utils"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

func Run2nodes(ctx context.Context, mockRendezvousString string) (*p2p.Node, *p2p.Node, *p2p.Node, *localcrypto.Keys, *localcrypto.Keys, *localcrypto.Keys, error) {
	mockbootstrapaddr := "/ip4/127.0.0.1/tcp/8520"
	mockbootstrapnodekeys, _, err := localcrypto.NewKeys()
	nodeopts := &options.NodeOptions{EnableNat: false}
	listenaddresses, _ := utils.StringsToAddrs([]string{mockbootstrapaddr})
	node, err := p2p.NewNode(ctx, nodeopts, true, nil, mockbootstrapnodekeys.PrivKey, connmgr.NewConnManager(1000, 50000, 30), listenaddresses, "")
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	mockbootstrapp2paddr := fmt.Sprintf("%s/p2p/%s", mockbootstrapaddr, node.Host.ID())
	log.Printf("bootstrap:%s", mockbootstrapp2paddr)

	bootstrapaddrs, _ := utils.StringsToAddrs([]string{mockbootstrapp2paddr})
	defaultnodeconfig := &cli.Config{RendezvousString: mockRendezvousString, BootstrapPeers: bootstrapaddrs}

	mockpeer1nodekeys, _, err := localcrypto.NewKeys()
	peer1listenaddresses, _ := utils.StringsToAddrs([]string{"/ip4/127.0.0.1/tcp/8551"})
	node1, err := p2p.NewNode(ctx, nodeopts, false, nil, mockpeer1nodekeys.PrivKey, connmgr.NewConnManager(10, 200, 60), peer1listenaddresses, "")
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	_ = node1.Bootstrap(ctx, *defaultnodeconfig)
	//log.Println("Announcing peer1...")
	//discovery.Advertise(ctx, node1.RoutingDiscovery, defaultnodeconfig.RendezvousString)
	//log.Println("Successfully announced peer1!")

	//TODO: use peerID to instead the RendezvousString, anyone can claim to this RendezvousString now"
	mockpeer2nodekeys, _, err := localcrypto.NewKeys()
	peer2listenaddresses, _ := utils.StringsToAddrs([]string{"/ip4/127.0.0.1/tcp/8552"})
	node2, err := p2p.NewNode(ctx, nodeopts, false, nil, mockpeer2nodekeys.PrivKey, connmgr.NewConnManager(10, 200, 60), peer2listenaddresses, "")
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	node2.Bootstrap(ctx, *defaultnodeconfig)
	return node, node1, node2, mockbootstrapnodekeys, mockpeer1nodekeys, mockpeer2nodekeys, nil
}

func RunNodesWithBootstrap(ctx context.Context, pidch chan int, n int) (string, []string, string, error) {
	var bootstrapaddr, testtempdir string
	peers := []string{}
	testtempdir, err := ioutil.TempDir("", "quorumtestdata")
	if err != nil {
		return "", []string{}, "", err
	}
	testconfdir := fmt.Sprintf("%s/%s", testtempdir, "config")
	testdatadir := fmt.Sprintf("%s/%s", testtempdir, "data")
	bootstrapport := 20666
	bootstrapapiport := 18010

	gopath := os.Getenv("GOROOT")
	if gopath == "" {
		gopath = build.Default.GOROOT
	}
	gocmd := gopath + "/bin/go"

	Fork(pidch, gocmd, "run", "main.go", "-bootstrap", "-listen", fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", bootstrapport), "-apilisten", fmt.Sprintf(":%d", bootstrapapiport), "-configdir", testconfdir, "-datadir", testdatadir)

	// wait bootstrap node
	checkctx, _ := context.WithTimeout(ctx, 60*time.Second)
	log.Printf("request: %s", fmt.Sprintf("https://127.0.0.1:%d", bootstrapapiport))
	result := CheckNodeRunning(checkctx, fmt.Sprintf("https://127.0.0.1:%d", bootstrapapiport))
	if result == false {
		return "", []string{}, "", fmt.Errorf("bootstrap node start failed")
	}
	bootstrapkeys, _ := localcrypto.LoadKeysFrom(testconfdir, "bootstrap", "txt")
	bootstrappeerid, err := peer.IDFromPublicKey(bootstrapkeys.PubKey)
	if err != nil {
		return "", []string{}, "", fmt.Errorf("can't load bootstrap keys:%s\n", err)
	}
	bootstrapaddr = fmt.Sprintf("/ip4/127.0.0.1/tcp/20666/p2p/%s", bootstrappeerid)
	log.Printf("bootstrap addr: %s\n", bootstrapaddr)

	// start other nodes
	peerport := 17001
	peerapiport := bootstrapapiport + 1
	i := 0
	for i < n {
		peerport = peerport + i
		peerapiport = peerapiport + i
		peername := fmt.Sprintf("peer%d", i+1)

		Fork(pidch, gocmd, "run", "main.go", "-peername", peername, "-listen", fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", peerport), "-apilisten", fmt.Sprintf(":%d", peerapiport), "-peer", bootstrapaddr, "-configdir", testconfdir, "-datadir", testdatadir)

		checkctx, _ = context.WithTimeout(ctx, 20*time.Second)
		result := CheckNodeRunning(checkctx, fmt.Sprintf("https://127.0.0.1:%d", peerapiport))
		if result == false {
			return "", []string{}, "", fmt.Errorf("%s node start failed", peername)
		}

		peerapiurl := fmt.Sprintf("https://127.0.0.1:%d", peerapiport)
		peers = append(peers, peerapiurl)

		i++
	}

	return bootstrapaddr, peers, testtempdir, nil
}

func Run2NodeProcessWith1Bootstrap(ctx context.Context, pidch chan int) (string, string, string, string, error) {
	testtempdir, err := ioutil.TempDir("", "quorumtestdata")
	if err != nil {
		return "", "", "", "", err
	}
	testconfdir := fmt.Sprintf("%s/%s", testtempdir, "config")
	testdatadir := fmt.Sprintf("%s/%s", testtempdir, "data")
	peer1apiport := ":18001"
	peer2apiport := ":18002"
	bootstrapapiport := ":18010"

	gopath := os.Getenv("GOROOT")
	if gopath == "" {
		gopath = build.Default.GOROOT
	}
	gocmd := gopath + "/bin/go"

	Fork(pidch, gocmd, "run", "main.go", "-bootstrap", "-listen", "/ip4/0.0.0.0/tcp/20666", "-apilisten", bootstrapapiport, "-configdir", testconfdir, "-datadir", testdatadir)

	checkctx, _ := context.WithTimeout(ctx, 60*time.Second)
	log.Printf("request: %s", fmt.Sprintf("https://127.0.0.1%s", bootstrapapiport))
	result := CheckNodeRunning(checkctx, fmt.Sprintf("https://127.0.0.1%s", bootstrapapiport))
	if result == false {
		return "", "", "", "", fmt.Errorf("bootstrap node start failed")
	}
	bootstrapkeys, _ := localcrypto.LoadKeysFrom(testconfdir, "bootstrap", "txt")
	bootstrappeerid, err := peer.IDFromPublicKey(bootstrapkeys.PubKey)
	if err != nil {
		return "", "", "", "", fmt.Errorf("can't load bootstrap keys:%s\n", err)
	}
	bootstrapaddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/20666/p2p/%s", bootstrappeerid)
	log.Printf("bootstrap addr: %s\n", bootstrapaddr)

	Fork(pidch, gocmd, "run", "main.go", "-peername", "peer1", "-listen", "/ip4/0.0.0.0/tcp/17001", "-apilisten", peer1apiport, "-peer", bootstrapaddr, "-configdir", testconfdir, "-datadir", testdatadir)
	Fork(pidch, gocmd, "run", "main.go", "-peername", "peer2", "-listen", "/ip4/0.0.0.0/tcp/17002", "-apilisten", peer2apiport, "-peer", bootstrapaddr, "-configdir", testconfdir, "-datadir", testdatadir)

	checkctx, _ = context.WithTimeout(ctx, 20*time.Second)
	resultpeer1 := CheckNodeRunning(checkctx, fmt.Sprintf("https://127.0.0.1%s", peer1apiport))
	if resultpeer1 == false {
		return "", "", "", "", fmt.Errorf("peer1 node start failed")
	}
	checkctx, _ = context.WithTimeout(ctx, 20*time.Second)
	resultpeer2 := CheckNodeRunning(checkctx, fmt.Sprintf("https://127.0.0.1%s", peer2apiport))
	if resultpeer2 == false {
		return "", "", "", "", fmt.Errorf("peer2 node start failed")
	}
	if resultpeer1 == true && resultpeer1 == true {
		log.Println("all set, testing start")
		return bootstrapaddr, fmt.Sprintf("https://127.0.0.1%s", peer1apiport), fmt.Sprintf("https://127.0.0.1%s", peer2apiport), testtempdir, nil
	}
	return "", "", "", "", fmt.Errorf("unknown error")
}

func CleanTestData(dir string) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	configdirexist := false
	datadirexist := false
	for _, file := range files {
		if file.Name() == "config" && file.IsDir() == true {
			configdirexist = true
		}
		if file.Name() == "data" && file.IsDir() == true {
			datadirexist = true
		}
	}
	if configdirexist == true && datadirexist == true {
		log.Printf("remove testdata:%s ...\n", dir)
		err = os.RemoveAll(dir)
		if err != nil {
			log.Printf("can't remove testdata:%s %s\n", dir, err)
		}
	} else {
		log.Printf("can't remove testdata:%s\n", dir)
	}
}

func Cleanup(dir string, peerapilist []string) {
	log.Printf("Clean testdata path: %s ...", dir)
	log.Println("peer api list", peerapilist)
	//add bootstrap node
	peerapilist = append(peerapilist, fmt.Sprintf("https://127.0.0.1:%d", 18010))
	for _, peerapi := range peerapilist {
		_, err := RequestAPI(peerapi, "/api/quit", "GET", "")
		if err == nil {
			log.Printf("kill node at %s ", peerapi)
		}

	}
	//waiting 3 sencodes for all processes quit.
	time.Sleep(3 * time.Second)

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	configdirexist := false
	datadirexist := false
	for _, file := range files {
		if file.Name() == "config" && file.IsDir() == true {
			configdirexist = true
		}
		if file.Name() == "data" && file.IsDir() == true {
			datadirexist = true
		}
	}

	if configdirexist == true && datadirexist == true {
		log.Printf("remove testdata:%s ...\n", dir)
		err = os.RemoveAll(dir)
		if err != nil {
			log.Printf("can't remove testdata:%s %s\n", dir, err)
		}
	} else {
		log.Printf("can't remove testdata:%s\n", dir)
	}
}
