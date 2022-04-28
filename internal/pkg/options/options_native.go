//go:build !js
// +build !js

package options

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/spf13/viper"
	//"path/filepath"
)

var nodeopts *NodeOptions
var nodeconfigdir string
var nodepeername string

const JWTKeyLength = 32
const defaultNetworkName = "nevis"
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
	v.Set("JWTKey", opt.JWTKey)
	v.Set("SelfJWTToken", opt.SelfJWTToken)
	v.Set("OthersJWTToken", opt.OthersJWTToken)
	return v.WriteConfig()
}

func (opt *NodeOptions) SetJWTKey(jwtKey string) error {
	opt.mu.Lock()
	defer opt.mu.Unlock()
	opt.JWTKey = jwtKey
	return opt.writeToconfig()
}

func (opt *NodeOptions) SetJWTToken(role, jwtToken string) error {
	opt.mu.Lock()
	defer opt.mu.Unlock()
	if role == "self" {
		opt.SelfJWTToken = jwtToken
	} else if role == "others" {
		opt.OthersJWTToken = jwtToken
	} else {
		return fmt.Errorf("un-support role: %s", role)
	}

	return opt.writeToconfig()
}

func (opt *NodeOptions) SetSignKeyMap(keyname, addr string) error {
	opt.mu.Lock()
	defer opt.mu.Unlock()
	opt.SignKeyMap[keyname] = addr
	return opt.writeToconfig()
}

func writeDefaultToconfig(v *viper.Viper) error {
	v.Set("EnableNat", true)
	v.Set("EnableRumExchange", false)
	v.Set("EnableDevNetwork", false)
	v.Set("NetworkName", defaultNetworkName)
	v.Set("MaxPeers", defaultMaxPeers)
	v.Set("ConnsHi", defaultConnsHi)
	v.Set("JWTKey", utils.GetRandomStr(JWTKeyLength))
	v.Set("JWTToken", "")
	v.Set("SignKeyMap", map[string]string{})
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

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			optionslog.Infof("config file not found, generating...")
			writeDefaultToconfig(v)
		} else {
			return nil, err
		}
	}

	jwtKey := v.GetString("JWTKey")
	if jwtKey == "" {
		jwtKey = utils.GetRandomStr(JWTKeyLength)
		v.Set("JWTKey", jwtKey)
		if err := v.WriteConfig(); err != nil {
			return nil, err
		}
	}

	for _, role := range []string{"self", "others"} {
		var configKey string
		if role == "self" {
			configKey = "SelfJWTToken"
		} else if role == "others" {
			configKey = "OthersJWTToken"
		}
		jwtToken := v.GetString(configKey)
		valid, _ := utils.IsJWTTokenValid(jwtToken, jwtKey)
		if jwtToken == "" || !valid {
			// HARDCOED: expire in 30 days
			exp := time.Now().Add(time.Hour * 24 * 30)
			jwtToken, err := utils.NewJWTToken(keyname, role, jwtKey, exp)
			if err != nil {
				return nil, err
			}

			v.Set(configKey, jwtToken)
			if err := v.WriteConfig(); err != nil {
				return nil, err
			}
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
	options.EnableRelay = v.GetBool("EnableRelay")
	options.EnableRelayService = v.GetBool("EnableRelayService")
	options.EnableNat = v.GetBool("EnableNat")
	options.EnableRumExchange = v.GetBool("EnableRumExchange")
	options.EnableDevNetwork = v.GetBool("EnableDevNetwork")
	options.NetworkName = v.GetString("NetworkName")
	if options.NetworkName == "" {
		options.NetworkName = defaultNetworkName
	}
	options.MaxPeers = v.GetInt("MaxPeers")
	if options.MaxPeers == 0 {
		options.MaxPeers = defaultMaxPeers
	}
	options.ConnsHi = v.GetInt("ConnsHi")
	if options.ConnsHi == 0 {
		options.ConnsHi = defaultConnsHi
	}

	options.SignKeyMap = v.GetStringMapString("SignKeyMap")
	options.JWTKey = v.GetString("JWTKey")
	options.SelfJWTToken = v.GetString("SelfJWTToken")
	options.OthersJWTToken = v.GetString("OthersJWTToken")

	return options, nil
}
