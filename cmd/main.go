package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	_ "github.com/golang/protobuf/ptypes/timestamp" //import for swaggo
	dsbadger2 "github.com/ipfs/go-ds-badger2"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	peerstore "github.com/libp2p/go-libp2p-core/peer"
	discovery "github.com/libp2p/go-libp2p-discovery"
	_ "github.com/multiformats/go-multiaddr" //import for swaggo
	"github.com/rumsystem/quorum/internal/pkg/api"
	"github.com/rumsystem/quorum/internal/pkg/appdata"
	"github.com/rumsystem/quorum/internal/pkg/chain"
	"github.com/rumsystem/quorum/internal/pkg/cli"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	"github.com/rumsystem/quorum/testnode"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/p2p"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	appapi "github.com/rumsystem/quorum/pkg/app/api"
	_ "google.golang.org/protobuf/proto" //import for swaggo

	//_ "google.golang.org/protobuf/proto/reflect/protoreflect" //import for swaggo
	_ "google.golang.org/protobuf/types/known/timestamppb" //import for swaggo

	"github.com/phayes/freeport"
)

const DEFAUT_KEY_NAME string = "default"

var (
	ReleaseVersion string
	GitCommit      string
	node           *p2p.Node
	signalch       chan os.Signal
	mainlog        = logging.Logger("main")
)

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

func createPubQueueDb(path string) (*storage.QSBadger, error) {
	var err error
	pubQueueDb := storage.QSBadger{}
	err = pubQueueDb.Init(path + "_pubqueue")
	if err != nil {
		return nil, err
	}

	return &pubQueueDb, nil
}

func saveLocalSeedsToAppdata(appdb *appdata.AppDb, dataDir string) {
	// NOTE: hardcode seed directory path
	seedPath := filepath.Join(filepath.Dir(dataDir), "seeds")
	if utils.DirExist(seedPath) {
		seeds, err := ioutil.ReadDir(seedPath)
		if err != nil {
			mainlog.Errorf("read seeds directory failed: %s", err)
		}

		for _, seed := range seeds {
			if seed.IsDir() {
				continue
			}

			path := filepath.Join(seedPath, seed.Name())
			seedByte, err := ioutil.ReadFile(path)
			if err != nil {
				mainlog.Errorf("read seed file failed: %s", err)
				continue
			}

			var seed handlers.GroupSeed
			if err := json.Unmarshal(seedByte, &seed); err != nil {
				mainlog.Errorf("unmarshal seed file failed: %s", err)
				continue
			}

			// if group seed already in app data then skip
			groupId := seed.GroupId
			savedSeed, err := appdb.GetGroupSeed(groupId)
			if err != nil {
				mainlog.Errorf("get group seed from appdb failed: %s", err)
				continue
			}
			if savedSeed != nil {
				// seed already exist, skip
				mainlog.Debugf("group id: %s, seed already exist, skip ...", groupId)
				continue
			}

			// save seed to app data
			pbSeed := handlers.ToPbGroupSeed(seed)
			err = appdb.SetGroupSeed(&pbSeed)
			if err != nil {
				mainlog.Errorf("save group seed failed: %s", err)
				continue
			}
		}
	}
}

func mainRet(config cli.Config) int {
	signalch = make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mainlog.Infof("Version: %s", GitCommit)
	peername := config.PeerName

	if config.IsBootstrap == true {
		peername = "bootstrap"
	}

	//Load node options
	nodeoptions, err := options.InitNodeOptions(config.ConfigDir, peername)
	if err != nil {
		cancel()
		mainlog.Fatalf(err.Error())
	}
	nodeoptions.IsRexTestMode = config.IsRexTestMode

	signkeycount, err := localcrypto.InitKeystore(config.KeyStoreName, config.KeyStoreDir)
	ksi := localcrypto.GetKeystore()
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
	ds, err := dsbadger2.NewDatastore(path.Join(config.DataDir, fmt.Sprintf("%s-%s", peername, "peerstore")), &dsbadger2.DefaultOptions)
	checkLockError(err)
	if err != nil {
		cancel()
		mainlog.Fatalf(err.Error())
	}

	if config.IsBootstrap == true {
		//bootstrop node connections: low watermarks: 1000  hi watermarks 50000, grace 30s
		node, err := p2p.NewNode(ctx, "", nodeoptions, config.IsBootstrap, ds, defaultkey, connmgr.NewConnManager(1000, 50000, 30), config.ListenAddresses, config.JsonTracer)

		if err != nil {
			mainlog.Fatalf(err.Error())
		}

		datapath := config.DataDir + "/" + config.PeerName
		dbManager, err := storage.CreateDb(datapath)
		if err != nil {
			mainlog.Fatalf(err.Error())
		}
		dbManager.TryMigration(0) //TOFIX: pass the node data_ver
		nodectx.InitCtx(ctx, "", node, dbManager, "pubsub", GitCommit)
		nodectx.GetNodeCtx().Keystore = ksi
		nodectx.GetNodeCtx().PublicKey = keys.PubKey
		nodectx.GetNodeCtx().PeerId = peerid

		mainlog.Infof("Host created, ID:<%s>, Address:<%s>", node.Host.ID(), node.Host.Addrs())
		h := &api.Handler{Node: node, NodeCtx: nodectx.GetNodeCtx(), GitCommit: GitCommit}
		go api.StartAPIServer(config, signalch, h, nil, node, nodeoptions, ks, ethaddr, true)
	} else {
		nodename := "default"
		//normal node connections: low watermarks: 10  hi watermarks 200, grace 60s
		node, err = p2p.NewNode(ctx, nodename, nodeoptions, config.IsBootstrap, ds, defaultkey, connmgr.NewConnManager(10, nodeoptions.ConnsHi, 60), config.ListenAddresses, config.JsonTracer)
		_ = node.Bootstrap(ctx, config)

		for _, addr := range node.Host.Addrs() {
			p2paddr := fmt.Sprintf("%s/p2p/%s", addr.String(), node.Host.ID())
			mainlog.Infof("Peer ID:<%s>, Peer Address:<%s>", node.Host.ID(), p2paddr)
		}

		//Discovery and Advertise had been replaced by PeerExchange
		mainlog.Infof("Announcing ourselves...")
		discovery.Advertise(ctx, node.RoutingDiscovery, config.RendezvousString)
		mainlog.Infof("Successfully announced!")

		peerok := make(chan struct{})
		go node.ConnectPeers(ctx, peerok, nodeoptions.MaxPeers, config)
		datapath := config.DataDir + "/" + config.PeerName
		dbManager, err := storage.CreateDb(datapath)
		if err != nil {
			mainlog.Fatalf(err.Error())
		}
		dbManager.TryMigration(0) //TOFIX: pass the node data_ver
		nodectx.InitCtx(ctx, nodename, node, dbManager, "pubsub", GitCommit)
		nodectx.GetNodeCtx().Keystore = ksi
		nodectx.GetNodeCtx().PublicKey = keys.PubKey
		nodectx.GetNodeCtx().PeerId = peerid

		//initial conn
		conn.InitConn()

		//initial group manager
		chain.InitGroupMgr()
		if nodeoptions.IsRexTestMode == true {
			chain.GetGroupMgr().SetRumExchangeTestMode()
		}

		// init the publish queue watcher
		doneCh := make(chan bool)
		pubqueueDb, err := createPubQueueDb(datapath)
		if err != nil {
			mainlog.Fatalf(err.Error())
		}
		chain.InitPublishQueueWatcher(doneCh, pubqueueDb)

		//load all groups
		err = chain.GetGroupMgr().LoadAllGroups()
		if err != nil {
			mainlog.Fatalf(err.Error())
		}

		//start sync all groups
		err = chain.GetGroupMgr().StartSyncAllGroups()
		if err != nil {
			mainlog.Fatalf(err.Error())
		}

		appdb, err := appdata.CreateAppDb(datapath)
		if err != nil {
			mainlog.Fatalf(err.Error())
		}
		checkLockError(err)

		// compatible with earlier versions: load group seeds and save to appdata
		saveLocalSeedsToAppdata(appdb, config.DataDir)

		//run local http api service
		h := &api.Handler{
			Node:      node,
			NodeCtx:   nodectx.GetNodeCtx(),
			Ctx:       ctx,
			GitCommit: GitCommit,
			Appdb:     appdb,
		}

		apiaddress := "https://%s/api/v1"
		if config.APIListenAddresses[:1] == ":" {
			apiaddress = fmt.Sprintf(apiaddress, "localhost"+config.APIListenAddresses)
		} else {
			apiaddress = fmt.Sprintf(apiaddress, config.APIListenAddresses)
		}
		appsync := appdata.NewAppSyncAgent(apiaddress, "default", appdb, dbManager)
		appsync.Start(10)
		apph := &appapi.Handler{
			Appdb:     appdb,
			Chaindb:   dbManager,
			GitCommit: GitCommit,
			Apiroot:   apiaddress,
			ConfigDir: config.ConfigDir,
			PeerName:  config.PeerName,
			NodeName:  nodectx.GetNodeCtx().Name,
		}
		go api.StartAPIServer(config, signalch, h, apph, node, nodeoptions, ks, ethaddr, false)
	}

	//attach signal
	signal.Notify(signalch, os.Interrupt, os.Kill, syscall.SIGTERM)
	signalType := <-signalch
	signal.Stop(signalch)

	if config.IsBootstrap != true {
		//Stop sync all groups
		chain.GetGroupMgr().StopSyncAllGroups()
		//teardown all groups
		chain.GetGroupMgr().TeardownAllGroups()
		//close ctx db
		nodectx.GetDbMgr().CloseDb()
	}

	//cleanup before exit
	mainlog.Infof("On Signal <%s>", signalType)
	mainlog.Infof("Exit command received. Exiting...")

	return 0
}

// @title Quorum Api
// @version 1.0
// @description Quorum Api Docs
// @BasePath /
func main() {
	if ReleaseVersion == "" {
		ReleaseVersion = "v1.0.0"
	}
	if GitCommit == "" {
		GitCommit = "devel"
	}
	utils.SetGitCommit(GitCommit)
	help := flag.Bool("h", false, "Display Help")
	version := flag.Bool("version", false, "Show the version")
	update := flag.Bool("update", false, "Update to the latest version")
	updateFrom := flag.String("from", "github", "Update from: github/qingcloud, default to github")

	// backup/restore flag
	isRestore := flag.Bool("restore", false, "restore the config, keystore and group seed")
	isBackup := flag.Bool("backup", false, "backup the config, keystore, group seed and group data")
	backupFile := flag.String("backup-file", "", "the backup file for restoring")
	password := flag.String("password", "", "the password for backuping/restoring")
	seedDir := flag.String("seeddir", "", "the group seed directory for restoring")

	config, err := cli.ParseFlags()

	chain.SetAutoAck(config.AutoAck)

	lvl, err := logging.LevelFromString("info")
	logging.SetAllLoggers(lvl)
	logging.SetLogLevel("appsync", "error")
	logging.SetLogLevel("appdata", "error")
	if err != nil {
		panic(err)
	}

	if config.IsDebug == true {
		logging.SetLogLevel("main", "debug")
		logging.SetLogLevel("crypto", "debug")
		logging.SetLogLevel("network", "debug")
		logging.SetLogLevel("autonat", "debug")
		logging.SetLogLevel("chain", "debug")
		logging.SetLogLevel("dbmgr", "debug")
		logging.SetLogLevel("chainctx", "debug")
		logging.SetLogLevel("syncer", "debug")
		logging.SetLogLevel("producer", "debug")
		logging.SetLogLevel("trxmgr", "debug")
		logging.SetLogLevel("conn", "debug")
		logging.SetLogLevel("rumexchange", "debug")
		logging.SetLogLevel("ssreceiver", "debug")
		logging.SetLogLevel("sssender", "debug")
		//logging.SetLogLevel("group", "debug")
		//logging.SetLogLevel("user", "debug")
		//logging.SetLogLevel("groupmgr", "debug")
		//logging.SetLogLevel("ping", "debug")
		//logging.SetLogLevel("chan", "debug")
		//logging.SetLogLevel("pubsub", "debug")
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
	if *update {
		err := errors.New(fmt.Sprintf("invalid `-from`: %s", *updateFrom))
		if *updateFrom == "qingcloud" {
			err = utils.CheckUpdateQingCloud(ReleaseVersion, "quorum")
		} else if *updateFrom == "github" {
			err = utils.CheckUpdate(ReleaseVersion, "quorum")
		}
		if err != nil {
			mainlog.Fatalf("Failed to do self-update: %s\n", err.Error())
		}
		return
	}

	if config.IsPing {
		if len(config.BootstrapPeers) == 0 {
			fmt.Println("Usage:", os.Args[0], "-ping", "-peer <peer> [-peer <peer> ...]")
			return
		}

		// FIXME: hardcode
		tcpAddr := "/ip4/127.0.0.1/tcp/0"
		wsAddr := "/ip4/127.0.0.1/tcp/0/ws"
		ctx := context.Background()
		node, err := libp2p.New(
			libp2p.ListenAddrStrings(tcpAddr, wsAddr),
			libp2p.Ping(false),
		)
		if err != nil {
			panic(err)
		}

		// configure our ping protocol
		pingService := &p2p.PingService{Host: node}
		node.SetStreamHandler(p2p.PingID, pingService.PingHandler)

		for _, addr := range config.BootstrapPeers {
			peer, err := peerstore.AddrInfoFromP2pAddr(addr)
			if err != nil {
				panic(err)
			}

			if err := node.Connect(ctx, *peer); err != nil {
				panic(err)
			}
			ch := pingService.Ping(ctx, peer.ID)
			fmt.Println()
			fmt.Println("pinging remote peer at", addr)
			for i := 0; i < 4; i++ {
				res := <-ch
				fmt.Println("PING", addr, "in", res.RTT)
			}
		}
		ping(config)
		return
	}

	if *isRestore {
		passwd, err := handlers.GetKeystorePassword(*password)
		if err != nil {
			mainlog.Fatalf("handlers.GetKeystorePassword failed: %s", err)
		}

		params := handlers.RestoreParam{
			Peername:    config.PeerName,
			BackupFile:  *backupFile,
			Password:    passwd,
			ConfigDir:   config.ConfigDir,
			KeystoreDir: config.KeyStoreDir,
			DataDir:     config.DataDir,
			SeedDir:     *seedDir,
		}
		restore(params)

		return
	}

	if *isBackup {
		handlers.Backup(config, *backupFile, *password)
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

func ping(config cli.Config) {
	if len(config.BootstrapPeers) == 0 {
		fmt.Println("Usage:", os.Args[0], "-ping", "-peer <peer> [-peer <peer> ...]")
		return
	}

	// FIXME: hardcode
	tcpAddr := "/ip4/127.0.0.1/tcp/0"
	wsAddr := "/ip4/127.0.0.1/tcp/0/ws"
	ctx := context.Background()
	node, err := libp2p.New(
		//ctx,
		libp2p.ListenAddrStrings(tcpAddr, wsAddr),
		libp2p.Ping(false),
	)
	if err != nil {
		panic(err)
	}

	// configure our ping protocol
	pingService := &p2p.PingService{Host: node}
	node.SetStreamHandler(p2p.PingID, pingService.PingHandler)

	for _, addr := range config.BootstrapPeers {
		peer, err := peerstore.AddrInfoFromP2pAddr(addr)
		if err != nil {
			panic(err)
		}

		if err := node.Connect(ctx, *peer); err != nil {
			panic(err)
		}
		ch := pingService.Ping(ctx, peer.ID)
		fmt.Println()
		fmt.Println("pinging remote peer at", addr)
		for i := 0; i < 4; i++ {
			res := <-ch
			fmt.Println("PING", addr, "in", res.RTT)
		}
	}
}

func restore(params handlers.RestoreParam) {
	if params.Peername == "" || params.BackupFile == "" || params.Password == "" || params.KeystoreDir == "" || params.ConfigDir == "" || params.SeedDir == "" || params.DataDir == "" {
		fmt.Printf("usage: %s -restore -peername=<peername> -backup-file=<backup-file-path> -password=<password> -keystoredir=<keystore-dir> -configdir=<config-dir> -seeddir=<seed-dir> -datadir=<data-dir>\n", os.Args[0])
		return
	}

	var err error
	params.BackupFile, err = filepath.Abs(params.BackupFile)
	if err != nil {
		mainlog.Fatalf("get absolute path for %s failed: %s", params.BackupFile, err)
	}
	params.ConfigDir, err = filepath.Abs(params.ConfigDir)
	if err != nil {
		mainlog.Fatalf("get absolute path for %s failed: %s", params.ConfigDir, err)
	}
	params.KeystoreDir, err = filepath.Abs(params.KeystoreDir)
	if err != nil {
		mainlog.Fatalf("get absolute path for %s failed: %s", params.KeystoreDir, err)
	}
	params.DataDir, _ = filepath.Abs(params.DataDir)
	if err != nil {
		mainlog.Fatalf("get absolute path for %s failed: %s", params.DataDir, err)
	}
	params.SeedDir, err = filepath.Abs(params.SeedDir)
	if err != nil {
		mainlog.Fatalf("get absolute path for %s failed: %s", params.SeedDir, err)
	}

	// go to restore directory before restore
	restoreDir := filepath.Dir(params.DataDir)
	if err := utils.EnsureDir(restoreDir); err != nil {
		mainlog.Fatalf("utils.EnsureDir(%s) failed: %s", restoreDir, err)
	}

	currentDir, err := os.Getwd()
	if err != nil {
		mainlog.Fatalf("os.Getwd failed: %s", err)
	}

	os.Chdir(restoreDir)
	defer os.Chdir(currentDir)

	handlers.Restore(params)

	var pidch chan int
	process := os.Args[0]

	apiPort, err := freeport.GetFreePort()
	if err != nil {
		mainlog.Fatalf("freeport.GetFreePort failed: %s", err)
	}
	testnode.Fork(
		pidch, params.Password, process,
		"-peername", params.Peername,
		"-apilisten", fmt.Sprintf(":%d", apiPort),
		"-configdir", params.ConfigDir,
		"-keystoredir", params.KeystoreDir,
		"-datadir", params.DataDir,
	)
	defer utils.RemoveAll("certs") // NOTE: HARDCODE

	peerBaseUrl := fmt.Sprintf("https://127.0.0.1:%d", apiPort)
	ctx := context.Background()
	checkctx, _ := context.WithTimeout(ctx, 300*time.Second)
	if ok := testnode.CheckApiServerRunning(checkctx, peerBaseUrl); !ok {
		mainlog.Fatal("api server start failed")
	}

	if utils.DirExist(params.SeedDir) {
		seeds, err := ioutil.ReadDir(params.SeedDir)
		if err != nil {
			mainlog.Errorf("read seeds directory failed: %s", err)
		}

		for _, seed := range seeds {
			if seed.IsDir() {
				continue
			}

			path := filepath.Join(params.SeedDir, seed.Name())
			seedByte, err := ioutil.ReadFile(path)
			if err != nil {
				mainlog.Errorf("read seed file failed: %s", err)
				continue
			}

			var seed handlers.GroupSeed
			if err := json.Unmarshal(seedByte, &seed); err != nil {
				mainlog.Errorf("unmarshal seed file failed: %s", err)
				continue
			}

			if _, err := api.JoinGroupByHTTPRequest(peerBaseUrl, seed); err != nil {
				mainlog.Errorf("join group %s failed: %s", seed.GroupId, err)
			}
		}
	}

	if _, _, err := testnode.RequestAPI(peerBaseUrl, "/api/quit", "GET", ""); err != nil {
		mainlog.Fatalf("quit app failed: %s", err)
	}
}
