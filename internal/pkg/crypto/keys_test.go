package crypto

import (
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"log"
	"math/rand"
	"testing"
)

func TestNewKeys(t *testing.T) {
	keys, _, err := NewKeys()
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

	keys1, _, err := NewKeys()

	keypkey, ok := keys.PrivKey.(*p2pcrypto.ECDSAPrivateKey)
	key1pkey, ok1 := keys1.PrivKey.(*p2pcrypto.ECDSAPrivateKey)
	if ok == ok1 == true && key1pkey.Equals(keypkey) {
		t.Errorf("error generated keys repeated.")
	}
}
