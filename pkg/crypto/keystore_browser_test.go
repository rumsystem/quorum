//go:build js && wasm
// +build js,wasm

package crypto

import (
	"testing"
)

func TestNewSignKey(t *testing.T) {
	password := "my.Passw0rd"
	ks, err := InitBrowserKeystore(password)
	if err != nil {
		t.Errorf("keystore init err: %s", err)
	}

	FactoryTestNewSignKey(ks, password, func(k Keystore, name string) (interface{}, error) {
		return k.(*BrowserKeystore).GetUnlockedKey(name)
	})(t)

}

func TestImportSignKey(t *testing.T) {
	password := "my.Passw0rd"
	ks, err := InitBrowserKeystore(password)
	if err != nil {
		t.Errorf("keystore init err: %s", err)
	}

	FactoryTestImportSignKey(ks)(t)
}

func TestNewEncryptKey(t *testing.T) {
	password := "my.Passw0rd"
	ks, err := InitBrowserKeystore(password)
	if err != nil {
		t.Errorf("keystore init err: %s", err)
	}

	FactoryTestNewEncryptKey(ks, password)(t)
}

func TestAlias(t *testing.T) {
	password := "my.Passw0rd"
	ks, err := InitBrowserKeystore(password)
	if err != nil {
		t.Errorf("keystore init err: %s", err)
	}

	FactoryTestAlias(ks, password, func(ks Keystore, alias string) string {
		kn, _ := ks.(*BrowserKeystore).GetKeynameFromAlias(alias)
		return kn
	})(t)
}

func TestEthSign(t *testing.T) {
	password := "my.Passw0rd"
	ks, err := InitBrowserKeystore(password)
	if err != nil {
		t.Errorf("keystore init err: %s", err)
	}

	FactoryTestEthSign(ks, password, func(k Keystore, name string) (interface{}, error) {
		return k.(*BrowserKeystore).GetUnlockedKey(name)
	})(t)
}
