package crypto

import (
	"bytes"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"log"
	"math/rand"
	"testing"
)

func TestNewKeys(t *testing.T) {
	keys, err := NewKeys()
	if err != nil {
		t.Errorf("Test New Keys err:%s", err)
	}

	testbytes := make([]byte, 128)
	rand.Read(testbytes)
	log.Printf("generate a random []byte: %x", testbytes)
	signedbytes, err := keys.PrivKey.Sign(testbytes)
	if err != nil {
		t.Errorf("Test RSA sign err: %s", err)
	}

	result, err := keys.PubKey.Verify(testbytes, signedbytes)
	if err != nil || result == false {
		t.Errorf("Test RSA sign verify err: %s", err)
	}

	keys1, err := NewKeys()
	pk1bytes, _ := p2pcrypto.MarshalPrivateKey(keys1.PrivKey)
	pkbytes, _ := p2pcrypto.MarshalPrivateKey(keys.PrivKey)
	if bytes.Equal(pk1bytes, pkbytes) {
		t.Errorf("error generated keys repeated.")
	}
}
