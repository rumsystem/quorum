package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	nodesdkapi "github.com/rumsystem/quorum/pkg/nodesdk/api"
	nodesdkdb "github.com/rumsystem/quorum/pkg/nodesdk/db"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

const DEFAUT_KEY_NAME string = "nodesdk_default"

var (
	ReleaseVersion string
	GitCommit      string
	signalch       chan os.Signal
	mainlog        = logging.Logger("nodesdk")
)

func main() {
	if ReleaseVersion == "" {
		ReleaseVersion = "v1.0.0"
	}
	if GitCommit == "" {
		GitCommit = "1.0.0" //"devel"
	}

	help := flag.Bool("h", false, "Display Help")
	version := flag.Bool("version", false, "Show the version")

	config, err := cli.ParseFlags()

	lvl, err := logging.LevelFromString("info")
	logging.SetAllLoggers(lvl)

	if err != nil {
		panic(err)
	}

	if config.IsDebug == true {
		logging.SetLogLevel("nodesdk", "debug")
	}

	if *help {
		fmt.Println("Output a help ")
		fmt.Println()
		fmt.Println("Usage:...")
		flag.PrintDefaults()
		return
	}

	if *version {
		fmt.Printf("%s - %s\n", ReleaseVersion, GitCommit)
		return
	}

	if err := utils.EnsureDir(config.DataDir); err != nil {
		panic(err)
	}

	_, _, err = utils.NewTLSCert()
	if err != nil {
		panic(err)
	}

	os.Exit(mainRet(config))
}

func mainRet(config cli.Config) int {
	mainlog.Infof("Version: %s", GitCommit)

	signalch = make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mainlog.Infof("Version: %s", GitCommit)
	peername := config.PeerName

	//Load node options
	nodeoptions, err := options.InitNodeOptions(config.ConfigDir, peername)
	if err != nil {
		cancel()
		mainlog.Fatalf(err.Error())
	}

	signkeycount, err := localcrypto.InitKeystore(config.KeyStoreName, config.KeyStoreDir)
	ksi := localcrypto.GetKeystore()
	if err != nil {
		cancel()
		mainlog.Fatalf(err.Error())
	}
	if err != nil {
		cancel()
		mainlog.Fatalf(err.Error())
	}

	ks, ok := ksi.(*localcrypto.DirKeyStore)
	if ok == false {
		//TODO: test other keystore type?
		//if there are no other keystores, exit and show error info.
		cancel()
		mainlog.Fatalf(err.Error())
	}

	password := os.Getenv("RUM_KSPASSWD")
	if signkeycount > 0 {
		if password == "" {
			password, err = localcrypto.PassphrasePromptForUnlock()
		}
		err = ks.Unlock(nodeoptions.SignKeyMap, password)
		if err != nil {
			mainlog.Fatalf(err.Error())
			cancel()
			return 0
		}
	} else {
		if password == "" {
			password, err = localcrypto.PassphrasePromptForEncryption()
			if err != nil {
				mainlog.Fatalf(err.Error())
				cancel()
				return 0
			}
			fmt.Println("Please keeping your password safe, We can't recover or reset your password.")
			fmt.Println("Your password:", password)
			fmt.Println("After saving the password, press any key to continue.")
			os.Stdin.Read(make([]byte, 1))
		}

		signkeyhexstr, err := localcrypto.LoadEncodedKeyFrom(config.ConfigDir, peername, "txt")
		if err != nil {
			cancel()
			mainlog.Fatalf(err.Error())
		}
		var addr string
		if signkeyhexstr != "" {
			addr, err = ks.Import(DEFAUT_KEY_NAME, signkeyhexstr, localcrypto.Sign, password)
		} else {
			addr, err = ks.NewKey(DEFAUT_KEY_NAME, localcrypto.Sign, password)
			if err != nil {
				mainlog.Fatalf(err.Error())
				cancel()
				return 0
			}
		}

		if addr == "" {
			mainlog.Fatalf("Load or create new signkey failed")
			cancel()
			return 0
		}
		err = nodeoptions.SetSignKeyMap(DEFAUT_KEY_NAME, addr)
		if err != nil {
			mainlog.Fatalf(err.Error())
			cancel()
			return 0
		}
		err = ks.Unlock(nodeoptions.SignKeyMap, password)
		if err != nil {
			mainlog.Fatalf(err.Error())
			cancel()
			return 0
		}

		fmt.Printf("load signkey: %d press any key to continue...\n", signkeycount)
	}

	_, err = ks.GetKeyFromUnlocked(localcrypto.Sign.NameString(DEFAUT_KEY_NAME))
	signkeycount = ks.UnlockedKeyCount(localcrypto.Sign)
	if signkeycount == 0 {
		mainlog.Fatalf("load signkey error, exit... %s", err)
		cancel()
		return 0
	}

	//Load default sign keys
	key, err := ks.GetKeyFromUnlocked(localcrypto.Sign.NameString(DEFAUT_KEY_NAME))

	defaultkey, ok := key.(*ethkeystore.Key)
	if ok == false {
		fmt.Println("load default key error, exit...")
		mainlog.Fatalf(err.Error())
		cancel()
		return 0
	}
	keys, err := localcrypto.SignKeytoPeerKeys(defaultkey)

	if err != nil {
		mainlog.Fatalf(err.Error())
		cancel()
		return 0
	}

	peerid, ethaddr, err := ks.GetPeerInfo(DEFAUT_KEY_NAME)
	if err != nil {
		cancel()
		mainlog.Fatalf(err.Error())
	}
	mainlog.Infof("eth addresss: <%s>", ethaddr)

	nodename := "nodesdk_default"

	datapath := config.DataDir + "/" + config.PeerName
	dbManager, err := nodesdkdb.CreateDb(datapath)
	if err != nil {
		mainlog.Fatalf(err.Error())
	}

	nodesdkctx.Init(ctx, nodename, dbManager, GitCommit)
	nodesdkctx.GetCtx().Keystore = ksi
	nodesdkctx.GetCtx().PublicKey = keys.PubKey
	nodesdkctx.GetCtx().PeerId = peerid

	//run local http api service
	nodeHandler := &nodesdkapi.NodeSDKHandler{
		NodeSdkCtx: nodesdkctx.GetCtx(),
		Ctx:        ctx,
		GitCommit:  GitCommit,
	}

	nodeApiAddress := "https://%s/api/v1"
	if config.NodeAPIListenAddress[:1] == ":" {
		nodeApiAddress = fmt.Sprintf(nodeApiAddress, "localhost"+config.NodeAPIListenAddress)
	} else {
		nodeApiAddress = fmt.Sprintf(nodeApiAddress, config.NodeAPIListenAddress)
	}

	//start node sdk server
	go nodesdkapi.StartNodeSDKServer(config, signalch, nodeHandler, nodeoptions)

	//attach signal
	signal.Notify(signalch, os.Interrupt, os.Kill, syscall.SIGTERM)
	signalType := <-signalch
	signal.Stop(signalch)

	nodesdkctx.GetDbMgr().CloseDb()

	//cleanup before exit
	mainlog.Infof("On Signal <%s>", signalType)
	mainlog.Infof("Exit command received. Exiting...")

	return 0
}

// reutrn EBUSY if LOCK is exist
func checkLockError(err error) {
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "Another process is using this Badger database.") {
			mainlog.Errorf(errStr)
			os.Exit(16)
		}
	}
}
