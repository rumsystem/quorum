package options

import (
	//"fmt"

	"path/filepath"

	"github.com/huo-ju/quorum/internal/pkg/utils"
	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/viper"
)

var optionslog = logging.Logger("options")

type NodeOptions struct {
	EnableNat bool
}

func writeDefaultToconfig() error {
	viper.Set("EnableNat", true)
	viper.SafeWriteConfig()
	return nil
}

func Load(dir string, keyname string) (*NodeOptions, error) {
	if err := utils.EnsureDir(dir); err != nil {
		optionslog.Errorf("check config directory failed: %s", err)
		return nil, err
	}

	viper.AddConfigPath(filepath.Dir(dir))
	viper.SetConfigName(keyname + "_options")
	viper.SetConfigType("toml")
	err := viper.ReadInConfig()
	if err != nil {
		optionslog.Infof("config file not found, generating...")
		writeDefaultToconfig()
		//_, ethkey, err := NewKeys()
		//if err != nil {
		//	return nil, err
		//}
		//err = ethkey.WritekeysToconfig()
		//if err != nil {
		//	return nil, err
		//}
	}
	err = viper.ReadInConfig()
	if err != nil {
		return nil, err
	}
	return &NodeOptions{EnableNat: true}, nil
}
