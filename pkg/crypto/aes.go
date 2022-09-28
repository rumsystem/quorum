package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

func CreateAesKey() ([]byte, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	return key, err
}

func AesEncrypt(data, key []byte) ([]byte, error) {
	cphr, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(cphr)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, data, nil), nil
}

func AesDecode(data, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcmDecrypt, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}
	nonceSize := gcmDecrypt.NonceSize()
	if len(data) < nonceSize {
		return nil, err
	}
	nonce, encrypteddata := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcmDecrypt.Open(nil, nonce, encrypteddata, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
