package crypto

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/glog"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

type Keys struct {
	PrivKey p2pcrypto.PrivKey
	PubKey  p2pcrypto.PubKey
}

type ethkey struct {
	privkey *ecdsa.PrivateKey
}

func NewKeys() (*Keys, *ethkey, error) {
	key, err := ethcrypto.GenerateKey()
	priv, pub, err := p2pcrypto.ECDSAKeyPairFromKey(key)
	if err != nil {
		return nil, nil, err
	}
	return &Keys{priv, pub}, &ethkey{key}, nil
}

func LoadKeys(dir string, keyname string) (*Keys, error) {
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
	viper.SetConfigName(keyname + "_keys")
	viper.SetConfigType("toml")
	err := viper.ReadInConfig()
	if err != nil {
		glog.Infof("Keys files not found, generating new keypair..")
		_, ethkey, err := NewKeys()
		if err != nil {
			return nil, err
		}
		err = ethkey.WritekeysToconfig()
		if err != nil {
			return nil, err
		}
	}
	err = viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	privstr := viper.GetString("priv")
	ethprivkey, err := ethcrypto.HexToECDSA(privstr)
	if err != nil {
		return nil, err
	}

	glog.Infof("Load keys from config")
	privkeybytes := ethcrypto.FromECDSA(ethprivkey)
	pubkeybytes := ethcrypto.FromECDSAPub(&ethprivkey.PublicKey)
	priv, err := p2pcrypto.UnmarshalSecp256k1PrivateKey(privkeybytes)
	pub, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)

	if err != nil {
		return nil, err
	}
	return &Keys{PrivKey: priv, PubKey: pub}, nil
}

func (key *ethkey) WritekeysToconfig() error {
	privkeybytes := ethcrypto.FromECDSA(key.privkey)
	if len(privkeybytes) == 0 {
		return fmt.Errorf("Private key encoding error")
	}
	viper.Set("priv", hex.EncodeToString(privkeybytes))
	viper.SafeWriteConfig()
	return nil
}
