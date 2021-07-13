package crypto

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	logging "github.com/ipfs/go-log/v2"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var cryptolog = logging.Logger("crypto")

type Keys struct {
	PrivKey p2pcrypto.PrivKey
	PubKey  p2pcrypto.PubKey
	EthAddr string
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

	address := ethcrypto.PubkeyToAddress(key.PublicKey).Hex()
	return &Keys{priv, pub, address}, &ethkey{key}, nil
}

func LoadKeysFrom(dir string, keyname string, filetype string) (*Keys, error) {
	keyfilepath := filepath.FromSlash(fmt.Sprintf("%s/%s_keys.%s", dir, keyname, filetype))
	keyhexstring := ""
	if filetype == "txt" {
		fmt.Println("Path: " + keyfilepath)
		f, err := os.Open(keyfilepath)
		if err != nil {
			_, ethkey, err := NewKeys()
			if err != nil {
				return nil, err
			}

			fmt.Println("call write keys to Path: " + keyfilepath)
			err = ethkey.WritekeysTo(keyfilepath, filetype)
			if err != nil {
				return nil, err
			}
			f, err = os.Open(keyfilepath)
			if err != nil {
				return nil, err
			}
		}
		defer f.Close()

		buf, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, err
		}

		keyhexstring = strings.TrimSpace(string(buf))
	} else {
		return nil, fmt.Errorf("unsupported filetype: %s", filetype)
	}

	ethprivkey, err := ethcrypto.HexToECDSA(keyhexstring)
	if err != nil {
		return nil, err
	}
	cryptolog.Infof("Load keys from config")
	privkeybytes := ethcrypto.FromECDSA(ethprivkey)
	pubkeybytes := ethcrypto.FromECDSAPub(&ethprivkey.PublicKey)
	priv, err := p2pcrypto.UnmarshalSecp256k1PrivateKey(privkeybytes)
	pub, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)

	if err != nil {
		return nil, err
	}

	address := ethcrypto.PubkeyToAddress(ethprivkey.PublicKey).Hex()
	return &Keys{PrivKey: priv, PubKey: pub, EthAddr: address}, nil
}

func (key *ethkey) WritekeysTo(filewithpath string, filetype string) error {
	privkeybytes := ethcrypto.FromECDSA(key.privkey)
	if len(privkeybytes) == 0 {
		return fmt.Errorf("Private key encoding error")
	}
	if filetype == "txt" {
		f, err := os.OpenFile(filewithpath, os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}
		defer f.Close()

		io.WriteString(f, hex.EncodeToString(privkeybytes))
	} else {
		return fmt.Errorf("unsupported filetype: %s", filetype)
	}
	return nil
}
