package nodesdkapi

import (
	"encoding/hex"

	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
)

func getEncryptData(data []byte, cipherKey string) ([]byte, error) {

	ciperKey, err := hex.DecodeString(cipherKey)
	if err != nil {
		return nil, err
	}

	encryptData, err := localcrypto.AesEncrypt(data, ciperKey)
	if err != nil {
		return nil, err
	}

	return encryptData, nil
}
