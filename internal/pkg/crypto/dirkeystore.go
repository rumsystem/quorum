//go:build !js
// +build !js

package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"filippo.io/age"
	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

type DirKeyStore struct {
	Name         string
	KeystorePath string
	password     string
	unlocked     map[string]interface{} //eth *Key or *X25519Identity, will be upgrade to generics
	signkeymap   map[string]string
	unlockTime   time.Time
	mu           sync.RWMutex
}

func InitDirKeyStore(name string, keydir string) (*DirKeyStore, int, error) {
	keydir, _ = filepath.Abs(keydir)

	_, err := os.Stat(keydir)
	if os.IsNotExist(err) {
		const dirPerm = 0700
		if err := os.MkdirAll(keydir, dirPerm); err != nil {
			return nil, 0, err
		}
	}

	signkeycount := 0
	files, err := ioutil.ReadDir(keydir)
	for _, f := range files {
		if strings.HasPrefix(f.Name(), Sign.Prefix()) == true {
			signkeycount++
		}
	}
	ks := &DirKeyStore{Name: name, KeystorePath: keydir, unlocked: make(map[string]interface{}), signkeymap: make(map[string]string)}
	return ks, signkeycount, nil
}

func (ks *DirKeyStore) UnlockedKeyCount(keytype KeyType) int {
	count := 0
	for k, _ := range ks.unlocked {
		if strings.HasPrefix(k, keytype.Prefix()) {
			count++
		}
	}
	return count
}

func (ks *DirKeyStore) Unlock(signkeymap map[string]string, password string) error {
	ks.signkeymap = signkeymap
	ks.password = password
	return nil
}

func (ks *DirKeyStore) Lock() error {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	for k, _ := range ks.unlocked {
		if strings.HasPrefix(k, Sign.Prefix()) { //zero the signkey in the memory
			signk, ok := ks.unlocked[k].(*ethkeystore.Key)
			if ok != true {
				return fmt.Errorf("The key %s is not a Sign key", k)
			}
			zeroSignKey(signk.PrivateKey)
			ks.unlocked[k] = nil
		}
		if strings.HasPrefix(k, Encrypt.Prefix()) {
			var zero = &age.X25519Identity{}
			ks.unlocked[k] = zero
		}
	}
	ks.unlocked = make(map[string]interface{})

	return nil
}

func (ks *DirKeyStore) GetPeerInfo(keyname string) (peerid peer.ID, ethaddr string, err error) {
	key, err := ks.GetKeyFromUnlocked(Sign.NameString(keyname))
	if err != nil {
		return "", "", err
	}
	signk, ok := key.(*ethkeystore.Key)
	if ok != true {
		return "", "", fmt.Errorf("The key %s is not a Sign key", keyname)
	}

	ethprivkey := signk.PrivateKey
	pubkeybytes := ethcrypto.FromECDSAPub(&ethprivkey.PublicKey)
	pub, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
	if err != nil {
		return "", "", err
	}

	peerid, err = peer.IDFromPublicKey(pub)
	if err != nil {
		return "", "", err
	}
	address := ethcrypto.PubkeyToAddress(ethprivkey.PublicKey).Hex()

	return peerid, address, nil
}

func (ks *DirKeyStore) GetKeyFromUnlocked(keyname string) (interface{}, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	if val, ok := ks.unlocked[keyname]; ok {
		return val, nil
	}

	//try unlock it
	if strings.HasPrefix(keyname, Sign.Prefix()) {
		addr := ks.signkeymap[keyname[len(Sign.Prefix()):]]
		if addr == "" {
			err := fmt.Errorf("can't find sign key %s addr", keyname)
			cryptolog.Warning(err)
			return nil, err
		}

		key, err := ks.LoadSignKey(keyname, common.HexToAddress(addr), ks.password)
		if err != nil {
			cryptolog.Warningf("key: %s can't be unlocked, err:%s", keyname, err)
			return nil, err
		}

		ks.unlocked[keyname] = key
		return ks.unlocked[keyname], nil

	} else if strings.HasPrefix(keyname, Encrypt.Prefix()) {
		key, err := ks.LoadEncryptKey(keyname, ks.password)
		if err == nil {
			ks.unlocked[keyname] = key
		} else {
			cryptolog.Warningf("key: %s can't be unlocked, err:%s", keyname, err)
			return nil, err
		}
		return ks.unlocked[keyname], nil
	}

	return nil, fmt.Errorf("key not exist or not be unlocked %s", keyname)
}

func JoinKeyStorePath(keysDirPath string, filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	}
	return filepath.Join(keysDirPath, filename)
}

func writeTemporaryKeyFile(file string, content []byte) (string, error) {
	// Create the keystore directory with appropriate permissions
	// in case it is not present yet.
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
		return "", err
	}
	// Atomic write: create a temporary hidden file first
	// then move it into place. TempFile assigns mode 0600.
	f, err := ioutil.TempFile(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
	if err != nil {
		return "", err
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	f.Close()
	return f.Name(), nil
}

func (ks *DirKeyStore) IfKeyExist(keyname string) (bool, error) {
	storefilename := JoinKeyStorePath(ks.KeystorePath, keyname)
	_, err := os.Stat(storefilename)
	if os.IsNotExist(err) {
		return false, nil
	}

	return true, err
}

func (ks *DirKeyStore) LoadEncryptKey(filename string, password string) (*age.X25519Identity, error) {
	storefilename := JoinKeyStorePath(ks.KeystorePath, filename)
	f, err := os.OpenFile(storefilename, os.O_RDONLY, 0600)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("key not exist.")

		}
		return nil, err
	}
	return AgeDecryptIdentityWithPassword(f, nil, password)
}

func (ks *DirKeyStore) LoadSignKey(filename string, addr common.Address, password string) (*ethkeystore.Key, error) {
	storefilename := JoinKeyStorePath(ks.KeystorePath, filename)
	return ks.getKey(addr, storefilename, password)
}

func (ks *DirKeyStore) StoreSignKey(filename string, key *ethkeystore.Key, password string) error {
	storefilename := JoinKeyStorePath(ks.KeystorePath, filename)
	keyjson, err := ethkeystore.EncryptKey(key, password, ethkeystore.StandardScryptN, ethkeystore.StandardScryptP)
	if err != nil {
		return err
	}
	// Write into temporary file
	tmpName, err := writeTemporaryKeyFile(storefilename, keyjson)
	if err != nil {
		return err
	}
	_, err = ks.getKey(key.Address, tmpName, password)
	if err != nil {
		msg := "An error was encountered when saving and verifying the keystore file. \n" +
			"This indicates that the keystore is corrupted. \n" +
			"The corrupted file is stored at \n%v\n" +
			"Please file a ticket at:\n\n" +
			"https://github.com/ethereum/go-ethereum/issues." +
			"The error was : %s"
		return fmt.Errorf(msg, tmpName, err)
	}
	return os.Rename(tmpName, storefilename)
}

func (ks *DirKeyStore) StoreEncryptKey(filename string, key *age.X25519Identity, password string) error {
	storefilename := JoinKeyStorePath(ks.KeystorePath, filename)

	r, err := age.NewScryptRecipient(password)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(storefilename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	return AgeEncrypt([]age.Recipient{r}, strings.NewReader(key.String()), f)
}

func (ks *DirKeyStore) getKey(addr common.Address, filename, auth string) (*ethkeystore.Key, error) {
	// Load the key from the keystore and decrypt its contents
	keyjson, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	key, err := ethkeystore.DecryptKey(keyjson, auth)
	if err != nil {
		return nil, err
	}
	// Make sure we're really operating on the requested key (no swap attacks)
	if key.Address != addr {
		return nil, fmt.Errorf("key content mismatch: have account %x, want %x", key.Address, addr)
	}
	return key, nil
}

func (ks *DirKeyStore) ImportEcdsaPrivKey(keyname string, privkey *ecdsa.PrivateKey, password string) (string, error) {
	exist, err := ks.IfKeyExist(keyname)
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
	err = ks.StoreSignKey(keyname, key, password)
	if err != nil {
		return "", err
	}
	return key.Address.String(), nil
}

func (ks *DirKeyStore) NewKeyWithDefaultPassword(keyname string, keytype KeyType) (string, error) {
	return ks.NewKey(keyname, keytype, ks.password)
}

func (ks *DirKeyStore) NewKey(keyname string, keytype KeyType, password string) (string, error) {
	//interface{} eth *PublicKey address or *X25519Recipient string, will be upgrade to generics

	keyname = keytype.NameString(keyname)
	exist, err := ks.IfKeyExist(keyname)
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
		err = ks.StoreEncryptKey(keyname, key, password)
		if err != nil {
			return "", err
		}

		ks.mu.Lock()
		defer ks.mu.Unlock()
		ks.unlocked[keyname] = key
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
		err = ks.StoreSignKey(keyname, key, password)
		if err != nil {
			return "", err
		}
		ks.mu.Lock()
		defer ks.mu.Unlock()
		ks.unlocked[keyname] = key
		return key.Address.String(), nil
	default:
		return "", fmt.Errorf("unsupported key type")
	}
}

func (ks *DirKeyStore) Import(keyname string, encodedkey string, keytype KeyType, password string) (string, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	cryptolog.Warningf("======= import key ==========")

	keyname = keytype.NameString(keyname)

	switch keytype {
	case Sign:
		privkey, err := ethcrypto.HexToECDSA(encodedkey)
		address, err := ks.ImportEcdsaPrivKey(keyname, privkey, password)
		if err == nil {
			cryptolog.Warningf("key %s imported, address: %s", keyname, address)
		}
		return address, err
	case Encrypt:
		key, err := age.ParseX25519Identity(encodedkey)
		if err != nil {
			return "", err
		}
		err = ks.StoreEncryptKey(keyname, key, password)
		if err != nil {
			return "", err
		}
		ks.unlocked[keyname] = key
		return key.Recipient().String(), nil

	}

	return "", nil
}

func (ks *DirKeyStore) Sign(data []byte, privKey p2pcrypto.PrivKey) ([]byte, error) {
	return privKey.Sign(data)
}

func (ks *DirKeyStore) SignByKeyName(keyname string, data []byte, opts ...string) ([]byte, error) {
	key, err := ks.GetKeyFromUnlocked(Sign.NameString(keyname))
	if err != nil {
		return nil, err
	}
	signk, ok := key.(*ethkeystore.Key)
	if ok != true {
		return nil, fmt.Errorf("The key %s is not a Sign key", keyname)
	}
	priv, _, err := p2pcrypto.ECDSAKeyPairFromKey(signk.PrivateKey)
	if err != nil {
		return nil, err
	}

	return priv.Sign(data)
	/*
		signature, signErr := priv.Sign(data)

		privByte, err := priv.Bytes()
		if err != nil {
			return nil, err
		}

		fmt.Printf("xxx signature: %s \nkeyname: %s \npriv: %s \ndata: %s\n", hex.EncodeToString(signature), keyname, hex.EncodeToString(privByte), hex.EncodeToString(data))

		return signature, signErr
	*/
}

func (ks *DirKeyStore) VerifySign(data, sig []byte, pubKey p2pcrypto.PubKey) (bool, error) {
	return pubKey.Verify(data, sig)
}

func (ks *DirKeyStore) VerifySignByKeyName(keyname string, data []byte, sig []byte, opts ...string) (bool, error) {
	key, err := ks.GetKeyFromUnlocked(Sign.NameString(keyname))
	if err != nil {
		return false, err
	}
	signk, ok := key.(*ethkeystore.Key)
	if ok != true {
		return false, fmt.Errorf("The key %s is not a Sign key", keyname)
	}
	_, pub, err := p2pcrypto.ECDSAKeyPairFromKey(signk.PrivateKey)
	if err != nil {
		return false, err
	}
	return pub.Verify(data, sig)
}

func (ks *DirKeyStore) GetEncodedPubkey(keyname string, keytype KeyType) (string, error) {
	ks.GetKeyFromUnlocked(keytype.NameString(keyname))
	if key, ok := ks.unlocked[keytype.NameString(keyname)]; ok {
		switch keytype {
		case Sign:
			signk, ok := key.(*ethkeystore.Key)
			if ok != true {
				return "", fmt.Errorf("The key %s is not a Sign key", keyname)
			}
			return hex.EncodeToString(ethcrypto.FromECDSAPub(&signk.PrivateKey.PublicKey)), nil
		case Encrypt:
			encryptk, ok := key.(*age.X25519Identity)
			if ok != true {
				return "", fmt.Errorf("The key %s is not a encrypt key", keyname)
			}
			return encryptk.Recipient().String(), nil
		}
		return "", fmt.Errorf("unknown keyType of %s", keyname)
	} else {
		return "", fmt.Errorf("key not exist :%s", keyname)
	}
}

func (ks *DirKeyStore) EncryptTo(to []string, data []byte) ([]byte, error) {
	recipients := []age.Recipient{}
	for _, key := range to {
		r, err := age.ParseX25519Recipient(key)
		if err != nil {
			return nil, err
		}
		recipients = append(recipients, r)
	}

	out := new(bytes.Buffer)
	err := AgeEncrypt(recipients, bytes.NewReader(data), out)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(out)
}

func (ks *DirKeyStore) Decrypt(keyname string, data []byte) ([]byte, error) {
	key, err := ks.GetKeyFromUnlocked(Encrypt.NameString(keyname))
	if err != nil {
		return nil, err
	}
	encryptk, ok := key.(*age.X25519Identity)

	if ok != true {
		return nil, fmt.Errorf("The key %s is not a encrypt key", keyname)
	}
	r, err := age.Decrypt(bytes.NewReader(data), encryptk)
	if err == nil {
		return ioutil.ReadAll(r)
	}
	return nil, err
}
