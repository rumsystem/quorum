//go:build !js
// +build !js

package options

import (
	"encoding/json"
	"sync"

	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/spf13/viper"
)

type RelayNodeOptions struct {
	ConfigDir   string
	PeerName    string
	NetworkName string
	SignKeyMap  map[string]string
	RC          relay.Resources `mapstructure:",remain"`
	mu          sync.RWMutex
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
	v.Set("NetworkName", defaultNetworkName)
	v.Set("SignKeyMap", map[string]string{})

	rc := relay.DefaultResources()
	rc.Limit = nil /* make it unlimit, so that it wont be a transient connection */
	rcMap := make(map[string]interface{})
	rcBytes, _ := json.Marshal(&rc)
	json.Unmarshal(rcBytes, &rcMap)

	v.Set("RC", rcMap)
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
	options.NetworkName = v.GetString("NetworkName")
	if options.NetworkName == "" {
		options.NetworkName = defaultNetworkName
	}
	options.SignKeyMap = v.GetStringMapString("SignKeyMap")

	rcIfc := v.Get("RC")
	if rcIfc != nil {
		err = v.UnmarshalKey("RC", &options.RC)
		if err != nil {
			return nil, err
		}
	} else {
		options.RC = relay.DefaultResources()
		options.RC.Limit = nil /* make it unlimit, so that it wont be a transient connection */
	}
	return options, nil
}
