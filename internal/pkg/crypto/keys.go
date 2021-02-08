package crypto

import (
	"crypto/rand"
	"github.com/golang/glog"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/spf13/viper"
	"path/filepath"
)

type Keys struct {
	PrivKey p2pcrypto.PrivKey
	PubKey  p2pcrypto.PubKey
}

func NewKeys(keyname string) (*Keys, error) {
	priv, pub, err := p2pcrypto.GenerateKeyPairWithReader(p2pcrypto.RSA, 4096, rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Keys{priv, pub}, nil
}

func LoadKeys(keyname string) (*Keys, error) {
	viper.AddConfigPath(filepath.Dir("./config/"))
	viper.SetConfigName(keyname + "_keys")
	viper.SetConfigType("toml")
	err := viper.ReadInConfig()
	if err != nil {
		glog.Infof("Keys files not found, generating new keypair..")
		newkeys, err := NewKeys(keyname)
		if err != nil {
			return nil, err
		}
		err = newkeys.WritekeysToconfig()
		if err != nil {
			return nil, err
		}
	}
	err = viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	privstr := viper.GetString("priv")
	pubstr := viper.GetString("pub")
	glog.Infof("Load keys from config")

	serializedpub, _ := p2pcrypto.ConfigDecodeKey(pubstr)
	pubfromconfig, err := p2pcrypto.UnmarshalPublicKey(serializedpub)
	if err != nil {
		return nil, err
	}

	serializedpriv, _ := p2pcrypto.ConfigDecodeKey(privstr)
	privfromconfig, err := p2pcrypto.UnmarshalPrivateKey(serializedpriv)
	if err != nil {
		return nil, err
	}
	return &Keys{PrivKey: privfromconfig, PubKey: pubfromconfig}, nil
}

func (keys *Keys) WritekeysToconfig() error {
	privkeybytes, err := p2pcrypto.MarshalPrivateKey(keys.PrivKey)
	if err != nil {
		return err
	}
	pubkeybytes, err := p2pcrypto.MarshalPublicKey(keys.PubKey)
	if err != nil {
		return err
	}
	viper.Set("priv", p2pcrypto.ConfigEncodeKey(privkeybytes))
	viper.Set("pub", p2pcrypto.ConfigEncodeKey(pubkeybytes))
	viper.SafeWriteConfig()
	return nil
}
