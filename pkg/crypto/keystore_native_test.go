//go:build !js
// +build !js

package crypto

import "testing"

func TestInitKeystore(t *testing.T) {
	keystoreName := "test-keystore"
	keystoreDir := t.TempDir()

	signkeycount, err := InitKeystore(keystoreName, keystoreDir)
	if err != nil {
		t.Fatalf("InitKeystore failed: %s", err)
	}

	if signkeycount != 0 {
		t.Fatal("InitKeystore failed: signkeycount != 0")
	}
}
