//go:build !js
// +build !js

package options

import (
	"fmt"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	//"path/filepath"
)

var logger = logging.Logger("options")
var nodeopts *NodeOptions
var nodeconfigdir string
var nodepeername string

const RumEnvPrefix = "RUM"
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

	if nodeopts.EnableDevNetwork {
		color.Red("WARNING! dev network mode is enabled!!!")
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
	err := initConfigfile(nodeconfigdir, nodepeername)
	if err != nil {
		return err
	}

	viper.Set("EnableNat", opt.EnableNat)
	viper.Set("EnableRumExchange", opt.EnableRumExchange)
	viper.Set("EnableDevNetwork", opt.EnableDevNetwork)
	viper.Set("SignKeyMap", opt.SignKeyMap)
	viper.Set("JWT", opt.JWT)

	return viper.WriteConfig()
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

func writeDefaultToconfig() error {
	return viper.SafeWriteConfig()
}

func initConfigfile(dir, keyname string) error {
	if dir == "" || keyname == "" {
		logger.Fatalf("config dir: %s or peername: %s is empty", dir, keyname)
	}
	if err := utils.EnsureDir(dir); err != nil {
		optionslog.Errorf("check config directory failed: %s", err)
		return err
	}

	viper.SetConfigFile(keyname + "_options.toml")
	viper.SetConfigName(keyname + "_options")
	viper.SetConfigType("toml")
	viper.AddConfigPath(dir)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			optionslog.Infof("config file not found, generating...")
			writeDefaultToconfig()
		} else {
			return err
		}
	}

	// get from environnent variable
	viper.SetEnvPrefix(RumEnvPrefix)
	viper.AutomaticEnv()
	viper.AllowEmptyEnv(true)

	// set default value
	viper.SetDefault("EnableNat", true)
	viper.SetDefault("EnableRumExchange", false)
	viper.SetDefault("EnableDevNetwork", false)
	viper.SetDefault("NetworkName", defaultNetworkName)
	viper.SetDefault("MaxPeers", defaultMaxPeers)
	viper.SetDefault("ConnsHi", defaultConnsHi)
	viper.SetDefault("SignKeyMap", map[string]string{})
	viper.SetDefault("JWT", JWT{
		Key:   utils.GetRandomStr(JWTKeyLength),
		Chain: &JWTListItem{},
		Node:  map[string]*JWTListItem{},
	})
	viper.SetDefault("EnableSnapshot", true)
	viper.SetDefault("EnablePubQue", true)

	return nil
}

func load(configdir, peername string) (*NodeOptions, error) {
	err := initConfigfile(configdir, peername)
	if err != nil {
		return nil, err
	}

	options := &NodeOptions{}
	if err := viper.Unmarshal(options); err != nil {
		panic(err)
	}

	return options, nil
}

func init() {
	// pflag.String("peername", "peer", "peername")
	// pflag.String("configdir", "./config/", "config and keys dir")
	// pflag.String("datadir", "./data/", "data dir")
	// pflag.String("keystoredir", "./keystore/", "keystore dir")
	// pflag.String("keystorename", "default", "keystore name")
	// pflag.String("keystorepass", "", "keystore password")
	// // pflag.Var("listen", "Adds a multiaddress to the listen list, e.g.: --listen /ip4/127.0.0.1/tcp/4215 --listen /ip/127.0.0.1/tcp/5215/ws")
	// pflag.String("apihost", "localhost", "api server ip or hostname")
	// pflag.Int("apiport", 5215, "api server listen port")
	// pflag.Var("peer", "bootstrap peer address")
	pflag.String("password", "", "keystore password")
	pflag.Bool("enablerelay", true, "enable relay")
	pflag.Bool("enablenat", true, "enable nat")
	pflag.Bool("enablerumexchange", true, "enable rumexchange")
	pflag.Bool("enabledevnetwork", true, "enable dev network")
	pflag.Bool("enablesnapshot", true, "enable snapshot")
	pflag.Bool("enablepubque", true, "enable pubque")
	pflag.Int("maxpeers", defaultMaxPeers, "max peer number")
	pflag.Int("connshi", defaultConnsHi, "max connshi")
	pflag.String("networkname", defaultNetworkName, "peer network name")
	// pflag.String("skippeers", "", "peer id lists, will be skipped in the pubsub connection")
	pflag.String("jsontracer", "", "output tracer data to a json file")

	// pflag.Parse()
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		logger.Fatalf("viper bind flags failed: %s", err)
	}
}

func NewViper() *viper.Viper {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvPrefix(RumEnvPrefix)

	return v
}
