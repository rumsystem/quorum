//go:build js && wasm
// +build js,wasm

package crypto

import (
	"bytes"
	"fmt"
	"strings"

	"filippo.io/age"
	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	quorumStorage "github.com/rumsystem/quorum/internal/pkg/storage"
)

type BrowserKeystore struct {
	store    *quorumStorage.QSIndexDB
	password string
}

func (ks *BrowserKeystore) Unlock(signkeymap map[string]string, password string) error {
	ks.password = password
	return nil
}

func (ks *BrowserKeystore) Lock() error {
	return nil
}

func (ks *BrowserKeystore) NewKey(keyname string, keytype KeyType) (string, error) {
	keyname = keytype.NameString(keyname)
	exist, err := ks.store.IsExist([]byte(keyname))
	if err != nil {
		return "", err
	}
	if exist == true {
		return "", fmt.Errorf("Key '%s' exists", keyname)
	}
	switch keytype {
	case Encrypt:
		key, err := age.GenerateX25519Identity()
		if err != nil {
			return "", err
		}
		err = ks.StoreEncryptKey(keyname, key)
		if err != nil {
			return "", err
		}

		return key.Recipient().String(), nil
	case Sign:
		privkey, err := ethcrypto.GenerateKey()
		if err != nil {
			return "", err
		}
		id, err := uuid.NewRandom()
		if err != nil {
			return "", err
		}
		key := &ethkeystore.Key{
			Id:         id,
			Address:    ethcrypto.PubkeyToAddress(privkey.PublicKey),
			PrivateKey: privkey,
		}
		err = ks.StoreSignKey(keyname, key)
		if err != nil {
			return "", err
		}
		return key.Address.String(), nil
	default:
		return "", fmt.Errorf("unsupported key type")
	}
}

func (ks *BrowserKeystore) Import(keyname string, encodedkey string, keytype KeyType) (string, error) {
	cryptolog.Warningf("======= import key ==========")

	keyname = keytype.NameString(keyname)

	switch keytype {
	case Sign:
		privkey, err := ethcrypto.HexToECDSA(encodedkey)
		exist, err := ks.store.IsExist([]byte(keyname))
		if err != nil {
			return "", err
		}
		if exist == true {
			return "", fmt.Errorf("Key '%s' exists", keyname)
		}
		id, err := uuid.NewRandom()
		if err != nil {
			return "", err
		}
		id, err = uuid.NewRandom()
		key := &ethkeystore.Key{
			Id:         id,
			Address:    ethcrypto.PubkeyToAddress(privkey.PublicKey),
			PrivateKey: privkey,
		}
		err = ks.StoreSignKey(keyname, key)
		if err != nil {
			return "", err
		}
		return key.Address.String(), nil
	case Encrypt:
		key, err := age.ParseX25519Identity(encodedkey)
		if err != nil {
			return "", err
		}
		err = ks.StoreEncryptKey(keyname, key)
		if err != nil {
			return "", err
		}
		return key.Recipient().String(), nil
	}

	return "", nil
}

// =============================== helpers
func (ks *BrowserKeystore) StoreEncryptKey(k string, key *age.X25519Identity) error {
	r, err := age.NewScryptRecipient(ks.password)
	if err != nil {
		return err
	}

	var b bytes.Buffer
	err = AgeEncrypt([]age.Recipient{r}, strings.NewReader(key.String()), &b)
	if err != nil {
		return err
	}

	return ks.store.Set([]byte(k), b.Bytes())
}

func (ks *BrowserKeystore) StoreSignKey(k string, key *ethkeystore.Key) error {
	enc, err := ethkeystore.EncryptKey(key, ks.password, ethkeystore.StandardScryptN, ethkeystore.StandardScryptP)
	if err != nil {
		return err
	}

	dec, err := ethkeystore.DecryptKey(enc, ks.password)
	if err != nil {
		return err
	}
	// Make sure we're really operating on the requested key (no swap attacks)
	if dec.Address != key.Address {
		return fmt.Errorf("key content mismatch: have account %x, want %x", dec.Address, key.Address)
	}
	if err != nil {
		msg := "An error was encountered when saving and verifying the keystore content. \n" +
			"This indicates that the keystore is corrupted. \n" +
			"Please file a ticket at:\n\n" +
			"https://github.com/ethereum/go-ethereum/issues." +
			"The error was : %s"
		return fmt.Errorf(msg, err)
	}

	return ks.store.Set([]byte(k), enc)
}
