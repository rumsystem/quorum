package crypto

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"filippo.io/age"
)

func AgeEncrypt(recipients []age.Recipient, in io.Reader, out io.Writer) error {
	w, err := age.Encrypt(out, recipients...)
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	if _, err := io.Copy(w, in); err != nil {
		return fmt.Errorf("%v", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("%v", err)
	}
	return err
}

// AgeDecrypt decrypt with the given password
func AgeDecrypt(password string, in io.Reader) (io.Reader, error) {
	identities := []age.Identity{
		&LazyScryptIdentity{password},
	}
	return age.Decrypt(in, identities...)
}

type LazyScryptIdentity struct {
	Password string
}

var _ age.Identity = &LazyScryptIdentity{}

func (i *LazyScryptIdentity) Unwrap(stanzas []*age.Stanza) (fileKey []byte, err error) {
	for _, s := range stanzas {
		if s.Type == "scrypt" && len(stanzas) != 1 {
			return nil, errors.New("an scrypt recipient must be the only one")
		}
	}
	if len(stanzas) != 1 || stanzas[0].Type != "scrypt" {
		return nil, age.ErrIncorrectIdentity
	}
	ii, err := age.NewScryptIdentity(i.Password)
	if err != nil {
		return nil, err
	}
	fileKey, err = ii.Unwrap(stanzas)
	if errors.Is(err, age.ErrIncorrectIdentity) {
		// ScryptIdentity returns ErrIncorrectIdentity for an incorrect
		// passphrase, which would lead Decrypt to returning "no identity
		// matched any recipient". That makes sense in the API, where there
		// might be multiple configured ScryptIdentity. Since in cmd/age there
		// can be only one, return a better error message.
		return nil, fmt.Errorf("incorrect passphrase")
	}
	return fileKey, err
}

func AgeDecryptIdentityWithPassword(in io.Reader, out io.Writer, password string) (*age.X25519Identity, error) {
	identities := []age.Identity{
		&LazyScryptIdentity{password},
	}
	r, err := age.Decrypt(in, identities...)
	if err != nil {
		return nil, err
	}
	keystr, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return age.ParseX25519Identity(string(keystr))
}
