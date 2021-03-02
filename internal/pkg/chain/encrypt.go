package chain

import ()

//sign
func RsaSignature(trx []byte) (signature []byte, err error) {

	/*
		hashed := sha256.Sum256(trx)
		signature, err := rsa.SignPKCS1v15(rand.Reader, GetContext().Privatekey, crypto.SHA256, hashed[:])

		if err != nil {
			return err
		}

		return signature
	*/

	var result []byte
	return result, nil
}

//verify
func RsaVerify(trx []byte, sig []byte) bool {
	/*
		hashed := sha256.Sum256(trx)
		return rsa.VerifyPKCS1v15(GetContext().PublicKey, crypto.SHA256, hashed, sig)
	*/

	return true
}
