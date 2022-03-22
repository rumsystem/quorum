package crypto

import (
	//"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/rumsystem/quorum/internal/pkg/logging"
)

var cryptolog = logging.Logger("crypto")

type Keys struct {
	PrivKey   p2pcrypto.PrivKey
	PubKey    p2pcrypto.PubKey
	EthAddr   string
	groupKeys map[string]*age.X25519Identity
}

func LoadEncodedKeyFrom(dir string, keyname string, filetype string) (string, error) {
	keyfilepath := filepath.FromSlash(fmt.Sprintf("%s/%s_keys.%s", dir, keyname, filetype))
	if filetype == "txt" {

		f, err := os.Open(keyfilepath)
		if err != nil {
			if os.IsNotExist(err) {
				return "", nil
			}
			return "", err
		}
		defer f.Close()

		buf, err := ioutil.ReadAll(f)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(buf)), nil
	} else {
		return "", fmt.Errorf("unsupported filetype: %s", filetype)
	}
}

func SignKeytoPeerKeys(key *ethkeystore.Key) (*Keys, error) {
	ethprivkey := key.PrivateKey
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

func Libp2pPubkeyToEthaddr(pubkey string) (string, error) {
	p2ppubkeybytes, err := p2pcrypto.ConfigDecodeKey(pubkey)
	if err != nil {
		return "", err
	}
	p2ppubkey, err := p2pcrypto.UnmarshalPublicKey(p2ppubkeybytes)
	if err != nil {
		return "", err
	}

	secp256k1pubkey, ok := p2ppubkey.(*p2pcrypto.Secp256k1PublicKey)
	if ok == true {
		btcecpubkey := (*btcec.PublicKey)(secp256k1pubkey)
		return ethcrypto.PubkeyToAddress(*btcecpubkey.ToECDSA()).Hex(), nil
	}
	return "", errors.New("input pubkey is not a Secp256k1PublicKey")
}
