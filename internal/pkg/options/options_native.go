//go:build !js
// +build !js

package options

import (
	"fmt"
	"path/filepath"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/spf13/viper"
	//"path/filepath"
)

var logger = logging.Logger("options")
var nodeopts *NodeOptions
var nodeconfigdir string
var nodepeername string

const JWTKeyLength = 32
const defaultNetworkName = "staten"
const defaultMaxPeers = 50
const defaultConnsHi = 100

func GetNodeOptions() *NodeOptions {
	return nodeopts
}

func InitNodeOptions(configdir, peername string) (*NodeOptions, error) {
	var err error
	nodeopts, err = load(configdir, peername)
	if err == nil {
		nodeconfigdir = configdir
		nodepeername = peername
	}
	return nodeopts, err
}

// GetConfigDir returns an absolute representation of path to the config directory
func GetConfigDir() (string, error) {
	if nodeconfigdir == "" {
		return "", fmt.Errorf("Please initConfigfile")
	}

	return filepath.Abs(nodeconfigdir)
}

func (opt *NodeOptions) writeToconfig() error {
	v, err := initConfigfile(nodeconfigdir, nodepeername)
	if err != nil {
		return err
	}

	v.Set("EnableNat", opt.EnableNat)
	v.Set("EnableRumExchange", opt.EnableRumExchange)
	v.Set("EnableDevNetwork", opt.EnableDevNetwork)
	v.Set("SignKeyMap", opt.SignKeyMap)
	v.Set("JWT", opt.JWT)
	return v.WriteConfig()
}

func (opt *NodeOptions) SetSignKeyMap(keyname, addr string) error {
	opt.mu.Lock()
	defer opt.mu.Unlock()
	opt.SignKeyMap[keyname] = addr
	return opt.writeToconfig()
}

func (opt *NodeOptions) DelSignKeyMap(keyname string) error {
	opt.mu.Lock()
	defer opt.mu.Unlock()
	delete(opt.SignKeyMap, keyname)
	return opt.writeToconfig()
}

func writeDefaultToconfig(v *viper.Viper) error {
	v.Set("EnableNat", true)
	v.Set("EnableRumExchange", false)
	v.Set("EnableDevNetwork", false)
	v.Set("NetworkName", defaultNetworkName)
	v.Set("MaxPeers", defaultMaxPeers)
	v.Set("ConnsHi", defaultConnsHi)
	v.Set("SignKeyMap", map[string]string{})
	v.Set("JWT", JWT{
		Key:   utils.GetRandomStr(JWTKeyLength),
		Chain: &JWTListItem{},
		Node:  map[string]*JWTListItem{},
	})
	return v.SafeWriteConfig()
}

func initConfigfile(dir string, keyname string) (*viper.Viper, error) {
	if err := utils.EnsureDir(dir); err != nil {
		optionslog.Errorf("check config directory failed: %s", err)
		return nil, err
	}

	v := viper.New()
	v.SetConfigFile(keyname + "_options.toml")
	v.SetConfigName(keyname + "_options")
	v.SetConfigType("toml")
	v.AddConfigPath(dir)
	v.SetEnvPrefix("RUM") // NOTE: hardcode
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			optionslog.Infof("config file not found, generating...")
			writeDefaultToconfig(v)
		} else {
			return nil, err
		}
	}

	return v, nil
}

func load(dir string, keyname string) (*NodeOptions, error) {
	v, err := initConfigfile(dir, keyname)
	if err != nil {
		return nil, err
	}

	err = v.ReadInConfig()
	if err != nil {
		return nil, err
	}

	options := &NodeOptions{}
	if err := v.Unmarshal(&options); err != nil {
		panic(err)
	}

	if options.JWT == nil {
		options.JWT = &JWT{
			Key:   utils.GetRandomStr(JWTKeyLength),
			Chain: &JWTListItem{},
			Node:  map[string]*JWTListItem{},
		}
	}
	if v.Get("EnableSnapshot") == nil {
		options.EnableSnapshot = true
	}

	if v.Get("EnablePubQue") == nil {
		options.EnablePubQue = true
	}

	if options.NetworkName == "" {
		options.NetworkName = defaultNetworkName
	}
	if options.MaxPeers == 0 {
		options.MaxPeers = defaultMaxPeers
	}
	if options.ConnsHi == 0 {
		options.ConnsHi = defaultConnsHi
	}

	return options, nil
}
