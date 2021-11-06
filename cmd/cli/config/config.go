package config

import (
	"bytes"
	"log"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/adrg/xdg"
	logging "github.com/ipfs/go-log/v2"
	qCrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
)

var signKeyMap = map[string]string{}

type QuorumConfig struct {
	Server               string
	ServerSSLCertificate string
	ServerSSLInsecure    bool
	KeyStoreName         string
	KeyStoreDir          string
	KeyStorePass         string
	JWT                  string
	MaxContentSize       int
	Muted                []string
}

type RumCliConfig struct {
	Quorum QuorumConfig
}

var RumConfig RumCliConfig
var Logger *logging.ZapEventLogger

func Init() {
	initLogger()
	configFilePath, err := xdg.ConfigFile("rumcli/config.toml")
	if err != nil {
		Logger.Fatal(err)
	}
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		// path/to/whatever does not exist
		f, err := os.OpenFile(configFilePath, os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			Logger.Fatal(err)
		}
		f.Close()
	}
	if _, err := toml.DecodeFile(configFilePath, &RumConfig); err != nil {
		Logger.Fatal(err)
	}

	initKeyStore()
}

func initLogger() {
	// init log file
	logFilePath, err := xdg.ConfigFile("rumcli/rumcli.log")
	if err != nil {
		log.Fatal(err)
	}
	// truncate error log on each starts
	f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	f.Close()
	if Logger == nil {
		cfg := logging.Config{
			Format: logging.PlaintextOutput,
			Stderr: false,
			Level:  logging.LevelInfo,
			File:   logFilePath,
		}
		logging.SetupLogging(cfg)
		Logger = logging.Logger("rumcli")
	}
}

func initKeyStore() {
	if RumConfig.Quorum.KeyStoreName != "" && RumConfig.Quorum.KeyStoreDir != "" {
		kCount, err := qCrypto.InitKeystore(RumConfig.Quorum.KeyStoreName, RumConfig.Quorum.KeyStoreDir)
		if err != nil {
			log.Fatal(err)
		}
		ksi := qCrypto.GetKeystore()
		ks, ok := ksi.(*qCrypto.DirKeyStore)
		if !ok {
			log.Fatal(err)
		}
		if kCount > 0 {
			password := RumConfig.Quorum.KeyStorePass
			if password == "" {
				password, err = qCrypto.PassphrasePromptForUnlock()
				if err != nil {
					log.Fatal(err)
				}
			}
			err = ks.Unlock(signKeyMap, password)
			if err != nil {
				log.Fatal(err)
			}
		}
		Logger.Infof("keystore OK, %d keys loaded", kCount)
	}
}

func Save() string {

	var configBuffer bytes.Buffer
	e := toml.NewEncoder(&configBuffer)
	if RumConfig.Quorum.MaxContentSize == 0 {
		// set default to 100
		RumConfig.Quorum.MaxContentSize = 100
	}
	err := e.Encode(RumConfig)
	if err != nil {
		Logger.Fatal(err)
	}

	configFilePath, err := xdg.ConfigFile("rumcli/config.toml")
	if err != nil {
		Logger.Fatal(err)
	}

	f, err := os.OpenFile(configFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		Logger.Fatal(err)
	}
	_, err = f.Write(configBuffer.Bytes())
	if err != nil {
		Logger.Fatal(err)
	}
	f.Close()
	return configFilePath
}
