//go:build !js
// +build !js

package crypto

import (
	"fmt"
	"testing"

	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
)

func TestLoadEncodedKeyFromEmptyDir(t *testing.T) {
	dir := t.TempDir()
	keyname := "xxxx"
	filetype := "txt"
	signkeyhexstr, err := LoadEncodedKeyFrom(dir, keyname, filetype)
	if err != nil {
		t.Fatalf("LoadEncodedKeyFrom failed: %s", err)
	}

	if len(signkeyhexstr) != 0 {
		t.Fatalf("LoadEncodedKeyFrom failed, signkeyhexstr should be empty")
	}
}

func TestLoadEncodedKeyFromNonTxtFile(t *testing.T) {
	name := "test-load-encoded-key"

	password := "my.Passw0rd"
	tempdir := fmt.Sprintf("%s/%s", t.TempDir(), name)
	dirks, _, err := InitDirKeyStore(name, tempdir)
	if err != nil {
		t.Fatalf("dirkeystore init failed: %s", err)
	}
	key1name := "test-key"
	keyname := Sign.NameString(key1name)
	_, err = dirks.NewKey(key1name, Sign, password)
	if err != nil {
		t.Fatalf("dirkeystore dirks.NewKey failed: %s", err)
	}

	_key1name, err := LoadEncodedKeyFrom(tempdir, keyname, "json")
	if err == nil {
		t.Fatalf("LoadEncodedKeyFrom should fail, but it didn't")
	}

	if len(_key1name) != 0 {
		t.Fatalf("LoadEncodedKeyFrom failed, _key1name should be empty")
	}
}

func TestSignKeytoPeerKeys(t *testing.T) {
	name := "test-signkey-to-peerkeys"

	password := "my.Passw0rd"
	tempdir := fmt.Sprintf("%s/%s", t.TempDir(), name)
	dirks, _, err := InitDirKeyStore(name, tempdir)
	if err != nil {
		t.Fatalf("dirkeystore init failed: %s", err)
	}
	key1name := "test-key"
	keyname := Sign.NameString(key1name)
	_, err = dirks.NewKey(key1name, Sign, password)
	if err != nil {
		t.Fatalf("dirkeystore dirks.NewKey failed: %s", err)
	}

	key, err := dirks.GetKeyFromUnlocked(keyname)
	if err != nil {
		t.Fatalf("dirkeystore dirks.GetKeyFromUnlocked failed: %s", err)
	}

	ethkey, ok := key.(*ethkeystore.Key)
	if !ok {
		t.Fatalf("key.(*ethkeystore.Key) failed: %s", err)
	}

	_, err = SignKeytoPeerKeys(ethkey)
	if err != nil {
		t.Fatalf("SignKeytoPeerKeys failed: %s", err)
	}
}
