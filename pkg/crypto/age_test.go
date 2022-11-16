package crypto

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"

	"filippo.io/age"
)

func TestAgeEncryptAndDecrypt(t *testing.T) {
	password := "My.passw0rd"
	recipient, err := age.NewScryptRecipient(password)
	if err != nil {
		t.Fatalf("age.NewScryptRecipient failed: %s", err)
	}

	plaintext := []byte("Hello, world!")
	encrypted := new(bytes.Buffer)

	if err := AgeEncrypt([]age.Recipient{recipient}, bytes.NewReader(plaintext), encrypted); err != nil {
		t.Fatalf("AgeEncrypt failed: %s", err)
	}

	decryptedReader, err := AgeDecrypt(password, encrypted)
	if err != nil {
		t.Fatalf("AgeDecrypt failed: %s", err)
	}

	decrypted, err := ioutil.ReadAll(decryptedReader)
	if err != nil {
		t.Fatalf("ioutil.ReadAll failed: %s", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("decrypted text does not match plaintext, got: %s expect: %s", decrypted, plaintext)
	}
}

func TestAgeDecryptIdentityWithPassword(t *testing.T) {
	password := "My.passw0rd"
	recipient, err := age.NewScryptRecipient(password)
	if err != nil {
		t.Fatalf("age.NewScryptRecipient failed: %s", err)
	}

	key, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("age.GenerateX25519Identity failed: %s", err)
	}

	output := new(bytes.Buffer)
	if err := AgeEncrypt([]age.Recipient{recipient}, strings.NewReader(key.String()), output); err != nil {
		t.Fatalf("AgeEncrypt failed: %s", err)
	}

	decKey, err := AgeDecryptIdentityWithPassword(output, nil, password)
	if err != nil {
		t.Fatalf("AgeDecryptIdentityWithPassword failed: %s", err)
	}

	if decKey == nil {
		t.Fatalf("AgeDecryptIdentityWithPassword returned nil key")
	}
}
