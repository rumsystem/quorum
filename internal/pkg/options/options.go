package options

import (
	//"fmt"
	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
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
	if dir[len(dir)-1:] != "/" && dir[len(dir)-1:] != "\\" { // add \\ for windows
		dir = dir + "/"
		if _, err := os.Stat(dir); err != nil {
			if os.IsNotExist(err) {
				err := os.Mkdir(dir, 0755)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
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
