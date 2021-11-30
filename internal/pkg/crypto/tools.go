package crypto

import (
	"crypto/sha256"
)

// Hash return the SHA256 checksum of the data
func Hash(data []byte) []byte {
	h := sha256.New()
	h.Write([]byte(data))
	hashed := h.Sum(nil)
	return hashed
}
