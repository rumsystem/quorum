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

var (
	_, b, _, _ = runtime.Caller(0)
	basepath   = filepath.Join(filepath.Dir(b), "../")
	logger     = logging.Logger("testnode")
)

const (
	KeystorePassword = "a_temp_password"
)

type Nodecliargs struct {
	Rextest bool
}

func RunNodesWithBootstrap(ctx context.Context, cli Nodecliargs, pidch chan int, n int) (string, []string, string, error) {
	var bootstrapaddr, testtempdir string
	peers := []string{}
	testtempdir, err := ioutil.TempDir("", "quorumtestdata")
	if err != nil {
		return "", []string{}, "", err
	}
	testconfdir := fmt.Sprintf("%s/%s", testtempdir, "config")
	testdatadir := fmt.Sprintf("%s/%s", testtempdir, "data")
	testkeystoredir := fmt.Sprintf("%s/%s", testtempdir, "keystore")
	bootstrapport := 20666
	bootstrapapiport := 18010

	gopath := os.Getenv("GOROOT")
	if gopath == "" {
		gopath = build.Default.GOROOT
	}
	gocmd := gopath + "/bin/go"

	if err := os.Chdir(basepath); err != nil {
		return "", []string{}, "", fmt.Errorf("os.Chdir(%s) failed: %s", basepath, err)
	}

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

	// start other nodes
	peerport := 17001
	peerapiport := bootstrapapiport + 1
	i := 0
	for i < n {
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

	return bootstrapBaseUrl, peers, testtempdir, nil
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

func Cleanup(dir string, peerapilist []string) {
	logger.Debugf("Clean testdata path: %s ...", dir)
	logger.Debug("peer api list", peerapilist)
	//add bootstrap node
	peerapilist = append(peerapilist, fmt.Sprintf("http://127.0.0.1:%d", 18010))
	for _, peerapi := range peerapilist {
		_, _, err := RequestAPI(peerapi, "/api/quit", "GET", "")
		if err == nil {
			logger.Debugf("kill node at %s ", peerapi)
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
