package crypto

import (
	"filippo.io/age"
	"io"
)

func EncryptDataForGroup(groupid string, dst io.Writer) (io.WriteCloser, error) {
	//TODO:
	//get recipients by groupid, recipients []age.Recipient
	//TEST ONLY mock key:
	//age1helmw6nufy93lcg5lylv2qxvdjdej8srcfk2hz3amw6lgms0td3qtw72fy
	mypub, err := age.ParseX25519Recipient("age1helmw6nufy93lcg5lylv2qxvdjdej8srcfk2hz3amw6lgms0td3qtw72fy")
	if err != nil {
		return nil, err
	}

	recipients := []age.Recipient{mypub}
	return age.Encrypt(dst, recipients...)
}
func DecryptDataForGroup(groupid string, src io.Reader) (io.Reader, error) {

	//TODO:
	//get my group public key
	//TEST ONLY mock key:
	//AGE-SECRET-KEY-1S77E2S2TF4SEVXTGJFQLN8NC6VG7TTLKYCNHSMA5CMYZAFY98NTQEN8QVV

	mykey, err := age.ParseX25519Identity("AGE-SECRET-KEY-1S77E2S2TF4SEVXTGJFQLN8NC6VG7TTLKYCNHSMA5CMYZAFY98NTQEN8QVV")
	if err != nil {
		return nil, err
	}
	return age.Decrypt(src, mykey)
}
