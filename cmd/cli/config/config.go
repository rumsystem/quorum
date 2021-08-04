package config

import (
	"bytes"
	"log"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/adrg/xdg"
)

type QuorumConfig struct {
	Server                  string
	ServerSSLCertificate    string
	ServerSSLCertificateKey string
	MaxContentSize          int
	Following               []string
}

type RumCliConfig struct {
	Quorum QuorumConfig
}

var RumConfig RumCliConfig

func Init() {
	configFilePath, err := xdg.ConfigFile("rumcli/config.toml")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		// path/to/whatever does not exist
		f, err := os.OpenFile(configFilePath, os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)
		}
		f.Close()
	}
	if _, err := toml.DecodeFile(configFilePath, &RumConfig); err != nil {
		log.Fatal(err)
	}
}

func Save() string {
	var configBuffer bytes.Buffer
	e := toml.NewEncoder(&configBuffer)
	if RumConfig.Quorum.MaxContentSize == 0 {
		// set default to 1000
		RumConfig.Quorum.MaxContentSize = 1000
	}
	err := e.Encode(RumConfig)
	if err != nil {
		log.Fatal(err)
	}

	configFilePath, err := xdg.ConfigFile("rumcli/config.toml")
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.OpenFile(configFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.Write(configBuffer.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	f.Close()
	return configFilePath
}
