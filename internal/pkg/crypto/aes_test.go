package crypto

import (
	"fmt"
	"testing"
	"time"
)

func TestAesEcryptAndDecrypt(t *testing.T) {
	key, err := CreateAesKey()
	if err != nil {
		t.Fatalf("CreateAesKey failed: %s", err)
	}

	plaintext := []byte(fmt.Sprintf("Hello, World! %s", time.Now()))
	ciphertext, err := AesEncrypt(plaintext, key)
	if err != nil {
		t.Fatalf("AesEncrypt failed: %s", err)
	}

	plaintext2, err := AesDecode(ciphertext, key)
	if err != nil {
		t.Fatalf("AesDecode failed: %s", err)
	}

	if string(plaintext) != string(plaintext2) {
		t.Fatalf("AesDecode failed, decrypt text is: %s, expect: %s", plaintext2, plaintext)
	}
}
