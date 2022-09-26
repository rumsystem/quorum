package crypto

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	p2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
)

// Hash return the SHA256 checksum of the data
func Hash(data []byte) []byte {
	h := sha256.New()
	h.Write([]byte(data))
	hashed := h.Sum(nil)
	return hashed
}

func Libp2pPubkeyToEthBase64(libp2ppubkey string) (string, error) {
	p2pkeyBytes, err := p2pcrypto.ConfigDecodeKey(libp2ppubkey)
	if err != nil {
		//is not a libp2pkey, may an ethkey?
		return libp2ppubkey, err
	}

	pubkey, err := p2pcrypto.UnmarshalPublicKey(p2pkeyBytes)
	if err != nil {
		return libp2ppubkey, err
	}

	secp256k1pubkey, ok := pubkey.(*p2pcrypto.Secp256k1PublicKey)
	if ok == true {
		spubkey := (*secp256k1.PublicKey)(secp256k1pubkey)
		return base64.RawURLEncoding.EncodeToString(ethcrypto.CompressPubkey(spubkey.ToECDSA())), nil
	}
	return libp2ppubkey, errors.New("convert to Secp256k1PublicKey failed")
}

func EthBase64ToLibp2pPubkey(ethbase64pubkey string) (string, error) {
	bytespubkey, err := base64.RawURLEncoding.DecodeString(ethbase64pubkey)
	if err != nil {
		return "", err
	}
	ecdsapubkey, err := ethcrypto.DecompressPubkey(bytespubkey)
	pubkeybytes := ethcrypto.FromECDSAPub(ecdsapubkey)
	p2ppubkey, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
	if err != nil {
		return "", err
	}
	SignPubkey, _ := p2pcrypto.MarshalPublicKey(p2ppubkey)
	pubkey := p2pcrypto.ConfigEncodeKey(SignPubkey)
	return pubkey, nil

}
