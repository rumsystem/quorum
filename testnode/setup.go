package testnode

import (
	"context"
	"fmt"
	"github.com/huo-ju/quorum/internal/pkg/cli"
	localcrypto "github.com/huo-ju/quorum/internal/pkg/crypto"
	"github.com/huo-ju/quorum/internal/pkg/p2p"
	"github.com/huo-ju/quorum/internal/pkg/utils"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	peer "github.com/libp2p/go-libp2p-core/peer"
	discovery "github.com/libp2p/go-libp2p-discovery"
	"io/ioutil"
	"log"
	"os"
	"syscall"
	"time"
)

func Run2nodes(ctx context.Context, mockRendezvousString string) (*p2p.Node, *p2p.Node, *p2p.Node, *localcrypto.Keys, *localcrypto.Keys, *localcrypto.Keys, error) {
	mockbootstrapaddr := "/ip4/127.0.0.1/tcp/8520"
	mockbootstrapnodekeys, err := localcrypto.NewKeys()
	listenaddresses, _ := utils.StringsToAddrs([]string{mockbootstrapaddr})
	node, err := p2p.NewNode(ctx, mockbootstrapnodekeys.PrivKey, connmgr.NewConnManager(1000, 50000, 30), listenaddresses, "")
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	mockbootstrapp2paddr := fmt.Sprintf("%s/p2p/%s", mockbootstrapaddr, node.Host.ID())
	log.Printf("bootstrap:%s", mockbootstrapp2paddr)

	bootstrapaddrs, _ := utils.StringsToAddrs([]string{mockbootstrapp2paddr})
	defaultnodeconfig := &cli.Config{RendezvousString: mockRendezvousString, BootstrapPeers: bootstrapaddrs}

	mockpeer1nodekeys, err := localcrypto.NewKeys()
	peer1listenaddresses, _ := utils.StringsToAddrs([]string{"/ip4/127.0.0.1/tcp/8551"})
	node1, err := p2p.NewNode(ctx, mockpeer1nodekeys.PrivKey, connmgr.NewConnManager(10, 200, 60), peer1listenaddresses, "")
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	_ = node1.Bootstrap(ctx, *defaultnodeconfig)
	log.Println("Announcing peer1...")
	discovery.Advertise(ctx, node1.RoutingDiscovery, defaultnodeconfig.RendezvousString)
	log.Println("Successfully announced peer1!")

	//TODO: use peerID to instead the RendezvousString, anyone can claim to this RendezvousString now"
	mockpeer2nodekeys, err := localcrypto.NewKeys()
	peer2listenaddresses, _ := utils.StringsToAddrs([]string{"/ip4/127.0.0.1/tcp/8552"})
	node2, err := p2p.NewNode(ctx, mockpeer2nodekeys.PrivKey, connmgr.NewConnManager(10, 200, 60), peer2listenaddresses, "")
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	node2.Bootstrap(ctx, *defaultnodeconfig)
	log.Println("Announcing peer2...")
	discovery.Advertise(ctx, node2.RoutingDiscovery, defaultnodeconfig.RendezvousString)
	log.Println("Successfully announced peer2")
	return node, node1, node2, mockbootstrapnodekeys, mockpeer1nodekeys, mockpeer2nodekeys, nil
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

	Fork(pidch, "/usr/bin/go", "run", "cmd/main.go", "-bootstrap", "-listen", "/ip4/0.0.0.0/tcp/20666", "-apilisten", bootstrapapiport, "-configdir", testconfdir, "-datadir", testdatadir)

	checkctx, _ := context.WithTimeout(ctx, 10*time.Second)
	result := CheckNodeRunning(checkctx, fmt.Sprintf("http://127.0.0.1%s", bootstrapapiport))
	if result == false {
		return "", "", "", "", fmt.Errorf("bootstrap node start failed")
	}
	bootstrapkeys, _ := localcrypto.LoadKeys(testconfdir, "bootstrap")
	bootstrappeerid, err := peer.IDFromPublicKey(bootstrapkeys.PubKey)
	if err != nil {
		return "", "", "", "", fmt.Errorf("can't load bootstrap keys:%s\n", err)
	}
	bootstrapaddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/20666/p2p/%s", bootstrappeerid)
	log.Printf("bootstrap addr: %s\n", bootstrapaddr)

	Fork(pidch, "/usr/bin/go", "run", "cmd/main.go", "-peername", "peer1", "-listen", "/ip4/0.0.0.0/tcp/17001", "-apilisten", peer1apiport, "-peer", bootstrapaddr, "-configdir", testconfdir, "-datadir", testdatadir)
	Fork(pidch, "/usr/bin/go", "run", "cmd/main.go", "-peername", "peer2", "-listen", "/ip4/0.0.0.0/tcp/17002", "-apilisten", peer2apiport, "-peer", bootstrapaddr, "-configdir", testconfdir, "-datadir", testdatadir)

	checkctx, _ = context.WithTimeout(ctx, 10*time.Second)
	resultpeer1 := CheckNodeRunning(checkctx, fmt.Sprintf("http://127.0.0.1%s", peer1apiport))
	if resultpeer1 == false {
		return "", "", "", "", fmt.Errorf("peer1 node start failed")
	}
	checkctx, _ = context.WithTimeout(ctx, 10*time.Second)
	resultpeer2 := CheckNodeRunning(checkctx, fmt.Sprintf("http://127.0.0.1%s", peer2apiport))
	if resultpeer2 == false {
		return "", "", "", "", fmt.Errorf("peer2 node start failed")
	}
	if resultpeer1 == true && resultpeer1 == true {
		log.Println("all set, testing start")
		return bootstrapaddr, fmt.Sprintf("http://127.0.0.1%s", peer1apiport), fmt.Sprintf("http://127.0.0.1%s", peer2apiport), testtempdir, nil
	}
	return "", "", "", "", fmt.Errorf("unknown error")
}

func Cleanup(dir string, pidlist []int) {
	log.Printf("Clean testdata path: %s ...", dir)
	log.Println("pidlist", pidlist)

	for _, pid := range pidlist {
		killpid := "N"
		log.Printf("Kill process: %d ? (Y/N)", pid)
		fmt.Scanf("%s\n", &killpid)
		if killpid == "Y" || killpid == "y" {
			syscall.Kill(pid, syscall.SIGKILL) //TODO: `go run` will start a child process to run the application, but SIGKILL can't kill child processes
			//pgid, err := syscall.Getpgid(pid)
			//if err == nil {
			//	syscall.Kill(-pgid, syscall.SIGKILL)
			//}

		}
	}

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
		fmt.Println(file.Name())
		fmt.Println()
	}

	deldir := "N"
	if configdirexist == true && datadirexist == true {
		fmt.Println("del: %s ?(Y/N)", dir)
		fmt.Scanf("%s\n", &deldir)
		if deldir == "Y" || deldir == "y" {
			err = os.RemoveAll(dir)
			fmt.Println("ok del this dir, result ", err)
		}
	}
}
