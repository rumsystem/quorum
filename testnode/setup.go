package testnode

import (
	"context"
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
)

const (
	NODE_LISTEN_PORT_INIT_VAL = 10000
	NODE_API_PORT_INIT_VAL    = 20000
	BOOTSTRAP_LISTEN_PORT     = 20666
	BOOTSTRAP_API_PORT        = 18010
)

var (
	_, b, _, _ = runtime.Caller(0)
	basepath   = filepath.Join(filepath.Dir(b), "../")
	logger     = logging.Logger("main_test")
)

const (
	KeystorePassword = "a_temp_password"
)

type Nodecliargs struct {
	Rextest bool
}

type NodeType int

const (
	BootstrapNode NodeType = iota
	FullNode
	ProducerNode
)

type NodeInfo struct {
	NodeName    string
	NodeType    NodeType
	PeerId      string
	DataDir     string
	ConfigDir   string
	KeystoreDir string
	APIBaseUrl  string
	ListenPort  int
	APIPort     int
	Addr        string
}

func RunNodesWithBootstrap(ctx context.Context, cli Nodecliargs, pidch chan int, fullnodenum int, bpnodenum int) ([]*NodeInfo, string, error) {
	var testtempdir string
	var bootstrapAddr string

	nodes := []*NodeInfo{}
	testtempdir, err := ioutil.TempDir("", "quorumtestdata")
	if err != nil {
		return nil, "", err
	}
	testconfdir := fmt.Sprintf("%s/%s", testtempdir, "config")
	testdatadir := fmt.Sprintf("%s/%s", testtempdir, "data")
	testkeystoredir := fmt.Sprintf("%s/%s", testtempdir, "keystore")

	gopath := os.Getenv("GOROOT")
	if gopath == "" {
		gopath = build.Default.GOROOT
	}
	gocmd := gopath + "/bin/go"

	if err := os.Chdir(basepath); err != nil {
		return nil, "", fmt.Errorf("os.Chdir(%s) failed: %s", basepath, err)
	}

	bootstrapNode := &NodeInfo{
		NodeName:    "bootstrap",
		NodeType:    BootstrapNode,
		ListenPort:  BOOTSTRAP_LISTEN_PORT,
		APIPort:     BOOTSTRAP_API_PORT,
		DataDir:     testdatadir,
		ConfigDir:   testconfdir,
		KeystoreDir: testkeystoredir,
	}

	nodes = append(nodes, bootstrapNode)

	listenPort := NODE_LISTEN_PORT_INIT_VAL
	apiPort := NODE_API_PORT_INIT_VAL
	i := 0

	for i < fullnodenum {
		nodename := fmt.Sprintf("fullnode_%d", i)
		nodekeystoredir := fmt.Sprintf("%s/%s_peer%s", testtempdir, "keystore", nodename)
		fullNode := &NodeInfo{
			NodeName:    nodename,
			NodeType:    FullNode,
			ListenPort:  listenPort,
			APIPort:     apiPort,
			DataDir:     testdatadir,
			ConfigDir:   testconfdir,
			KeystoreDir: nodekeystoredir,
		}
		nodes = append(nodes, fullNode)
		listenPort += 1
		apiPort += 1
		i++
	}

	i = 0
	for i < bpnodenum {
		nodename := fmt.Sprintf("producernode_%d", i)
		nodekeystoredir := fmt.Sprintf("%s/%s_peer%s", testtempdir, "keystore", nodename)
		producerNode := &NodeInfo{
			NodeName:    nodename,
			NodeType:    ProducerNode,
			ListenPort:  listenPort,
			APIPort:     apiPort,
			DataDir:     testdatadir,
			ConfigDir:   testconfdir,
			KeystoreDir: nodekeystoredir,
		}
		nodes = append(nodes, producerNode)
		listenPort += 1
		apiPort += 1
		i++
	}

	for _, node := range nodes {

		logger.Debugf("Try create node %s", node.NodeName)

		switch node.NodeType {
		case BootstrapNode:
			Fork(pidch, KeystorePassword, gocmd, "run", "main.go",
				"bootstrapnode",
				"--listen", fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", node.ListenPort),
				"--apiport", fmt.Sprintf("%d", node.APIPort),
				"--configdir", node.ConfigDir,
				"--keystoredir", node.KeystoreDir,
				"--datadir", node.DataDir)

		case FullNode:
			Fork(pidch, KeystorePassword, gocmd, "run", "main.go",
				"fullnode",
				"--peername", node.NodeName,
				"--listen", fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", node.ListenPort),
				"--apiport", fmt.Sprintf("%d", node.APIPort),
				"--peer", bootstrapAddr,
				"--configdir", testconfdir,
				"--keystoredir", node.KeystoreDir,
				"--datadir", testdatadir,
				fmt.Sprintf("--rextest=%s", strconv.FormatBool(cli.Rextest)))

		case ProducerNode:
			Fork(pidch, KeystorePassword, gocmd, "run", "main.go",
				"producernode",
				"--peername", node.NodeName,
				"--listen", fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", node.ListenPort),
				"--apiport", fmt.Sprintf("%d", node.APIPort),
				"--peer", bootstrapAddr,
				"--configdir", testconfdir,
				"--keystoredir", node.KeystoreDir,
				"--datadir", testdatadir)
		}

		node.APIBaseUrl = fmt.Sprintf("http://127.0.0.1:%d", node.APIPort)

		checkctx, _ := context.WithTimeout(ctx, 60*time.Second)
		peerId, result := CheckNodeRunning(checkctx, node.APIBaseUrl)

		if !result {
			return nil, "", fmt.Errorf("node <%s> start failed", node.NodeName)
		}

		node.Addr = fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", node.ListenPort, peerId)
		logger.Debugf("Node <%s> addr: <%s>, started", node.NodeName, node.Addr)

		if node.NodeType == BootstrapNode {
			bootstrapAddr = node.Addr
		}
	}

	/*
		Fork(pidch, KeystorePassword, gocmd, "run", "main.go", "bootstrapnode", "--listen", fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", bootstrapport), "--apiport", fmt.Sprintf("%d", bootstrapapiport), "--configdir", testconfdir, "--keystoredir", testkeystoredir, "--datadir", testdatadir)

		// wait bootstrap node
		bootstrapBaseUrl := fmt.Sprintf("http://127.0.0.1:%d", bootstrapapiport)
		checkctx, _ := context.WithTimeout(ctx, 60*time.Second)
		logger.Debugf("request: %s", bootstrapBaseUrl)
		bootstrappeerid, result := CheckNodeRunning(checkctx, bootstrapBaseUrl)
		if result == false {
			return "", []string{}, "", fmt.Errorf("bootstrap node start failed")
		}
		bootstrapaddr = fmt.Sprintf("/ip4/127.0.0.1/tcp/20666/p2p/%s", bootstrappeerid)
		logger.Debugf("bootstrap addr: %s\n", bootstrapaddr)
	*/
	/*
		peerport := 17001
		peerapiport := bootstrapapiport + 1
		i := 0

		// start users nodes
		for i < fullnodenum {
			peerport = peerport + i
			peerapiport = peerapiport + i
			peername := fmt.Sprintf("peer%d", i+1)
			testpeerkeystoredir := fmt.Sprintf("%s/%s_peer%s", testtempdir, "keystore", peername)
			Fork(pidch, KeystorePassword, gocmd, "run", "main.go", "fullnode", "--peername", peername, "--listen", fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", peerport), "--apiport", fmt.Sprintf("%d", peerapiport), "--peer", bootstrapaddr, "--configdir", testconfdir, "--keystoredir", testpeerkeystoredir, "--datadir", testdatadir, fmt.Sprintf("--rextest=%s", strconv.FormatBool(cli.Rextest)))

			checkctx, _ = context.WithTimeout(ctx, 60*time.Second)
			_, result := CheckNodeRunning(checkctx, fmt.Sprintf("http://127.0.0.1:%d", peerapiport))
			if result == false {
				return "", []string{}, "", fmt.Errorf("%s node start failed", peername)
			}

			peerapiurl := fmt.Sprintf("http://127.0.0.1:%d", peerapiport)
			peers = append(peers, peerapiurl)

			i++
		}

		// start bp nodes
		for i < bpnodenum {
			//TODO: run bp nodes
		}

	*/
	return nodes, testtempdir, nil
}

func newSignKeyfromKeystore(keyname string, ks *localcrypto.DirKeyStore) {
}

func newEncryptKeyfromKeystore(keyname string, ks *localcrypto.DirKeyStore) {
}

func newKeystore(ksdir string) (*localcrypto.DirKeyStore, bool) {
	signkeycount, err := localcrypto.InitKeystore("default", ksdir)
	ksi := localcrypto.GetKeystore()
	if err != nil {
		return nil, false
	}

	ks, ok := ksi.(*localcrypto.DirKeyStore)
	if ok == false {
		return nil, false
	}

	if signkeycount == 0 {
		return ks, true
	}
	return nil, false
}

func CleanTestData(dir string) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		logger.Fatal(err)
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
		logger.Debugf("remove testdata: %s ...", dir)
		err = os.RemoveAll(dir)
		if err != nil {
			logger.Errorf("can't remove testdata: %s %s", dir, err)
		}
	} else {
		logger.Warnf("can't remove testdata: %s", dir)
	}
}

func Cleanup(dir string, nodes []*NodeInfo /*peerapilist []string */) {
	logger.Debugf("Clean testdata path: %s ...", dir)
	//logger.Debug("peer api list", peerapilist)
	//add bootstrap node
	//peerapilist = append(peerapilist, fmt.Sprintf("http://127.0.0.1:%d", 18010))
	/*
		for _, peerapi := range peerapilist {
			_, _, err := RequestAPI(peerapi, "/api/quit", "GET", "")
			if err == nil {
				logger.Debugf("kill node at %s ", peerapi)
			}
		}
	*/

	for _, node := range nodes {
		_, _, err := RequestAPI(node.APIBaseUrl, "/api/quit", "GET", "")
		if err == nil {
			logger.Debugf("kill node %s ", node.NodeName)
		}
	}

	//waiting 3 sencodes for all processes quit.
	time.Sleep(3 * time.Second)

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		logger.Fatal(err)
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
		logger.Debugf("remove testdata:%s ...", dir)
		err = os.RemoveAll(dir)
		if err != nil {
			logger.Warnf("can't remove testdata: %s %s", dir, err)
		}
	} else {
		logger.Warnf("can't remove testdata: %s", dir)
	}
}
