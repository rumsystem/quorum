package chain

import ()

//sign
func Sign(trx []byte) ([]byte, error) {
	signature, err := GetChainCtx().Privatekey.Sign(trx)
	return signature, err
}

//verify
func Verify(data, sign []byte) (bool, error) {
	verify, err := GetChainCtx().PublicKey.Verify(data, sign)
	return verify, err
}
