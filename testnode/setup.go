package testnode

import (
	"context"
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	//localcrypto "github.com/rumsystem/quorum/pkg/crypto"
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
		NodeName:   "bootstrap",
		NodeType:   BootstrapNode,
		ListenPort: BOOTSTRAP_LISTEN_PORT,
		APIPort:    BOOTSTRAP_API_PORT,
	}
	bootstrapNode.ConfigDir = filepath.Join(testconfdir, bootstrapNode.NodeName)
	bootstrapNode.DataDir = filepath.Join(testdatadir, bootstrapNode.NodeName)
	bootstrapNode.KeystoreDir = filepath.Join(testkeystoredir, bootstrapNode.NodeName)

	nodes = append(nodes, bootstrapNode)

	listenPort := NODE_LISTEN_PORT_INIT_VAL
	apiPort := NODE_API_PORT_INIT_VAL

	i := 0
	for i < fullnodenum {
		fullNode := &NodeInfo{
			NodeName:   fmt.Sprintf("fullnode_%d", i),
			NodeType:   FullNode,
			ListenPort: listenPort,
			APIPort:    apiPort,
		}
		fullNode.ConfigDir = filepath.Join(testconfdir, fullNode.NodeName)
		fullNode.DataDir = filepath.Join(testdatadir, fullNode.NodeName)
		fullNode.KeystoreDir = filepath.Join(testkeystoredir, fullNode.NodeName)
		nodes = append(nodes, fullNode)
		listenPort += 1
		apiPort += 1
		i++
	}

	i = 0
	for i < bpnodenum {
		producerNode := &NodeInfo{
			NodeName:   fmt.Sprintf("producernode_%d", i),
			NodeType:   ProducerNode,
			ListenPort: listenPort,
			APIPort:    apiPort,
		}
		producerNode.ConfigDir = filepath.Join(testconfdir, producerNode.NodeName)
		producerNode.DataDir = filepath.Join(testdatadir, producerNode.NodeName)
		producerNode.KeystoreDir = filepath.Join(testkeystoredir, producerNode.NodeName)
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
				"--datadir", testdatadir)

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

	return nodes, testtempdir, nil
}

func CleanTestData(dir string) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		logger.Fatal(err)
	}
	configdirexist := false
	datadirexist := false
	for _, file := range files {
		if file.Name() == "config" && file.IsDir() {
			configdirexist = true
		}
		if file.Name() == "data" && file.IsDir() {
			datadirexist = true
		}
	}
	if configdirexist && datadirexist {
		logger.Debugf("remove testdata: %s ...", dir)
		err = os.RemoveAll(dir)
		if err != nil {
			logger.Errorf("can't remove testdata: %s %s", dir, err)
		}
	} else {
		logger.Warnf("can't remove testdata: %s", dir)
	}
}

func Cleanup(dir string, nodes []*NodeInfo) {
	logger.Debug("Try kill all running nodes")
	for _, node := range nodes {
		_, _, err := RequestAPI(node.APIBaseUrl, "/api/quit", "GET", "")
		if err == nil {
			logger.Debugf("kill node %s ", node.NodeName)
		}
	}

	//waiting 3 sencodes for all processes quit.
	time.Sleep(3 * time.Second)

	logger.Debugf("Clean testdata path: %s ...", dir)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		logger.Fatal(err)
	}

	configdirexist := false
	datadirexist := false
	for _, file := range files {
		if file.Name() == "config" && file.IsDir() {
			configdirexist = true
		}
		if file.Name() == "data" && file.IsDir() {
			datadirexist = true
		}
	}

	if configdirexist && datadirexist {
		logger.Debugf("remove testdata:%s ...", dir)
		err = os.RemoveAll(dir)
		if err != nil {
			logger.Warnf("can't remove testdata: %s %s", dir, err)
		}
	} else {
		logger.Warnf("can't remove testdata: %s", dir)
	}
}

/*
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
*/
