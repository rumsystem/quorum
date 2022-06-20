//go:build !js
// +build !js

package options

import (
	"sync"

	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/spf13/viper"
)

type RelayNodeOptions struct {
	ConfigDir  string
	PeerName   string
	SignKeyMap map[string]string
	mu         sync.RWMutex
}

func InitRelayNodeOptions(configdir, peername string) (*RelayNodeOptions, error) {
	nodeopts, err := loadRelayNodeOptions(configdir, peername)
	nodeopts.ConfigDir = configdir
	nodeopts.PeerName = peername

	return nodeopts, err
}

func (opt *RelayNodeOptions) writeToconfig() error {
	v, err := initRelayNodeConfigfile(opt.ConfigDir, opt.PeerName)
	if err != nil {
		return err
	}
	v.Set("SignKeyMap", opt.SignKeyMap)
	return v.WriteConfig()
}

func (opt *RelayNodeOptions) SetSignKeyMap(keyname, addr string) error {
	opt.mu.Lock()
	defer opt.mu.Unlock()
	opt.SignKeyMap[keyname] = addr
	return opt.writeToconfig()
}

func (opt *RelayNodeOptions) DelSignKeyMap(keyname string) error {
	opt.mu.Lock()
	defer opt.mu.Unlock()
	delete(opt.SignKeyMap, keyname)
	return opt.writeToconfig()
}

func initRelayNodeConfigfile(dir string, keyname string) (*viper.Viper, error) {
	if err := utils.EnsureDir(dir); err != nil {
		optionslog.Errorf("check config directory failed: %s", err)
		return nil, err
	}

	v := viper.New()
	v.SetConfigFile(keyname + "_options.toml")
	v.SetConfigName(keyname + "_options")
	v.SetConfigType("toml")
	v.AddConfigPath(dir)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			optionslog.Infof("config file not found, generating...")
			writeDefaultRelayNodeConfig(v)
		} else {
			return nil, err
		}
	}
	return v, nil
}

func writeDefaultRelayNodeConfig(v *viper.Viper) error {
	v.Set("SignKeyMap", map[string]string{})
	return v.SafeWriteConfig()
}

func loadRelayNodeOptions(dir string, keyname string) (*RelayNodeOptions, error) {
	v, err := initRelayNodeConfigfile(dir, keyname)
	if err != nil {
		return nil, err
	}
	err = v.ReadInConfig()
	if err != nil {
		return nil, err
	}

	options := &RelayNodeOptions{}
	options.SignKeyMap = v.GetStringMapString("SignKeyMap")

	return options, nil
}
