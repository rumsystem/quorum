//go:build !js
// +build !js

package crypto

import (
	"fmt"
	"testing"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/options"
)

var (
	logger = logging.Logger("crypto")
)

func TestInitDirKeyStore(t *testing.T) {
	name := "testkeystore"
	tempdir := fmt.Sprintf("%s/%s", t.TempDir(), name)
	logger.Debugf("tempdir %s", tempdir)
	_, count, err := InitDirKeyStore(name, tempdir)
	if err != nil {
		t.Errorf("keystore init err: %s", err)
	}
	if count != 0 {
		t.Errorf("init new keystore count should be 0, not : %d", count)
	}
}

func TestNewSignKey(t *testing.T) {
	name := "testnewkey"
	password := "my.Passw0rd"
	tempdir := fmt.Sprintf("%s/%s", t.TempDir(), name)
	dirks, _, err := InitDirKeyStore(name, tempdir)
	if err != nil {
		t.Errorf("keystore init err: %s", err)
	}

	FactoryTestNewSignKey(dirks, password, func(k Keystore, name string) (interface{}, error) {
		return k.(*DirKeyStore).GetKeyFromUnlocked(name)
	})(t)

}

func TestImportSignKey(t *testing.T) {
	name := "testnewkey"
	tempdir := fmt.Sprintf("%s/%s", t.TempDir(), name)
	logger.Debugf("tempdir %s", tempdir)
	dirks, _, err := InitDirKeyStore(name, tempdir)
	if err != nil {
		t.Errorf("keystore init err: %s", err)
	}
	FactoryTestImportSignKey(dirks)(t)
}

func TestNewEncryptKey(t *testing.T) {
	name := "testnewkey"
	password := "my.Passw0rd"
	tempdir := fmt.Sprintf("%s/%s", t.TempDir(), name)
	dirks, _, err := InitDirKeyStore(name, tempdir)
	if err != nil {
		t.Errorf("keystore init err: %s", err)
	}
	FactoryTestNewEncryptKey(dirks, password)(t)
}

func TestAlias(t *testing.T) {
	nodeoptions, err := options.InitNodeOptions(t.TempDir(), "testpeername")
	name := "testnewkey"
	password := "my.Passw0rd"
	tempdir := fmt.Sprintf("%s/%s", t.TempDir(), name)
	t.Log(tempdir)
	dirks, _, err := InitDirKeyStore(name, tempdir)
	if err != nil {
		t.Errorf("keystore init err: %s", err)
	}
	//unlock keymap
	err = dirks.Unlock(nodeoptions.SignKeyMap, password)
	if err != nil {
		t.Errorf("keystore usnlock err: %s", err)
	}

	FactoryTestAlias(dirks, password, func(ks Keystore, alias string) string {
		return ks.(*DirKeyStore).AliasToKeyname(alias)
	})(t)
}

func TestEthSign(t *testing.T) {
	name := "testnewkey"
	password := "my.Passw0rd"
	tempdir := fmt.Sprintf("%s/%s", t.TempDir(), name)
	dirks, _, err := InitDirKeyStore(name, tempdir)
	if err != nil {
		t.Errorf("keystore init err: %s", err)
	}

	FactoryTestEthSign(dirks, password, func(k Keystore, name string) (interface{}, error) {
		return k.(*DirKeyStore).GetKeyFromUnlocked(name)
	})(t)
}

func TestEthSignByKeyName(t *testing.T) {
	name := "testethsignbyname"
	password := "my.Passw0rd"
	tempdir := fmt.Sprintf("%s/%s", t.TempDir(), name)
	ks, _, err := InitDirKeyStore(name, tempdir)
	if err != nil {
		t.Errorf("keystore init err: %s", err)
	}

	keyname := "key1"
	_, err = ks.NewKey(keyname, Sign, password)

	testdata := "some random text for testing"
	testdatahash := Hash([]byte(testdata))
	sig, err := ks.EthSignByKeyName(keyname, testdatahash)
	if err != nil {
		t.Errorf("eth sign by keyname failed: %s", err)
	}
	verifyresult, err := ks.EthVerifyByKeyName(keyname, testdatahash, sig)
	if err != nil {
		t.Errorf("eth verify signature by keyname failed: %s", err)
	}
	if verifyresult == false {
		t.Errorf("eth verify signature by keyname failure")
	}

	// verifyresult is false
	testdata = "new random text for testing"
	testdatahash = Hash([]byte(testdata))
	verifyresult, err = ks.EthVerifyByKeyName(keyname, testdatahash, sig)
	if err != nil {
		t.Errorf("eth verify signature by keyname failed: %s", err)
	}
	if verifyresult == true {
		t.Errorf("eth verify signature by keyname failure")
	}
}
