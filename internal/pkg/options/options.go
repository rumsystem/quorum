package options

import (
	//"fmt"

	"github.com/huo-ju/quorum/internal/pkg/utils"
	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/viper"
)

var optionslog = logging.Logger("options")

const JWTKeyLength = 32

type NodeOptions struct {
	EnableNat bool
	JWTToken  string
	JWTKey    string
}

func writeDefaultToconfig(v *viper.Viper) error {
	v.Set("EnableNat", true)
	v.Set("JWTKey", utils.GetRandomStr(JWTKeyLength))
	v.Set("JWTToken", "")
	return v.SafeWriteConfig()
}

func SetJWTKey(dir, keyname, jwtKey string) error {
	v, err := initConfig(dir, keyname)
	if err != nil {
		return err
	}

	v.Set("JWTKey", jwtKey)
	return v.WriteConfig()
}

func SetJWTToken(dir, keyname, jwtToken string) error {
	v, err := initConfig(dir, keyname)
	if err != nil {
		return err
	}

	v.Set("JWTToken", jwtToken)
	return v.WriteConfig()
}

func initConfig(dir string, keyname string) (*viper.Viper, error) {
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

	if v.GetString("JWTKey") == "" {
		v.Set("JWTKey", utils.GetRandomStr(JWTKeyLength))
		if err := v.WriteConfig(); err != nil {
			return nil, err
		}
	}

	return v, nil
}

func Load(dir string, keyname string) (*NodeOptions, error) {
	v, err := initConfig(dir, keyname)
	if err != nil {
		return nil, err
	}

	options := new(NodeOptions)
	if err := v.Unmarshal(options); err != nil {
		return nil, err
	}

	return options, nil
}
