package crypto

import (
	"crypto/ecdsa"
	"fmt"

	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

type KeyType int

const (
	Encrypt KeyType = iota
	Sign
)

var ks Keystore

//singlaton
func GetKeystore() Keystore {
	return ks
}

func zeroSignKey(k *ecdsa.PrivateKey) {
	b := k.D.Bits()
	for i := range b {
		b[i] = 0
	}
}

func (kt KeyType) Prefix() string {
	switch kt {
	case Encrypt:
		return "encrypt_"
	case Sign:
		return "sign_"
	}
	return ""
}

func (kt KeyType) NameString(keyname string) string {
	switch kt {
	case Encrypt:
		return fmt.Sprintf("encrypt_%s", keyname)
	case Sign:
		return fmt.Sprintf("sign_%s", keyname)
	}
	return ""
}

type Keystore interface {
	Unlock(signkeymap map[string]string, password string) error
	Lock() error
	NewKey(keyname string, keytype KeyType, password string) (string, error)
	NewKeyWithDefaultPassword(keyname string, keytype KeyType) (string, error)
	Import(keyname string, encodedkey string, keytype KeyType, password string) (string, error)
	Sign(data []byte, privKey p2pcrypto.PrivKey) ([]byte, error)
	VerifySign(data, signature []byte, pubKey p2pcrypto.PubKey) (bool, error)
	SignByKeyName(keyname string, data []byte, opts ...string) ([]byte, error)
	VerifySignByKeyName(keyname string, data []byte, sig []byte, opts ...string) (bool, error)
	EncryptTo(to []string, data []byte) ([]byte, error)
	Decrypt(keyname string, data []byte) ([]byte, error)
	GetEncodedPubkey(keyname string, keytype KeyType) (string, error)
	GetPeerInfo(keyname string) (peerid peer.ID, ethaddr string, err error)
}
