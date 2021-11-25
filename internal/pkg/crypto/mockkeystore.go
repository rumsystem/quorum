//go:build !js
// +build !js

package crypto

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"filippo.io/age"
	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/spf13/viper"
)

type MockKeyStore struct {
	Name         string
	KeystorePath string
	v            *viper.Viper
	keys         map[string]string
	unlocked     map[string]interface{} //eth *Key or *X25519Identity, will be upgrade to generics
	unlockTime   time.Time
	mu           sync.RWMutex
}

func InitMockKeyStore(name string, keydir string) (*MockKeyStore, int, error) {
	keydir, _ = filepath.Abs(keydir)

	_, err := os.Stat(keydir)
	if os.IsNotExist(err) {
		const dirPerm = 0700
		if err := os.MkdirAll(keydir, dirPerm); err != nil {
			return nil, 0, err
		}
	}
	v, err := initConfigfile(keydir, name)
	if err != nil {
		return nil, 0, err
	}

	signkeycount := 0
	allkeys, err := load(keydir, name)
	if err == nil {
		signkeycount = len(allkeys)
	}
	ks := &MockKeyStore{Name: name, KeystorePath: keydir, unlocked: make(map[string]interface{}), v: v, keys: allkeys}
	return ks, signkeycount, nil
}

func (ks *MockKeyStore) Unlock(signkeymap map[string]string, password string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	for name, val := range ks.keys {
		if strings.HasPrefix(name, Encrypt.Prefix()) == true {
			key, err := age.ParseX25519Identity(val)
			if err == nil {
				ks.unlocked[name] = key
			} else {
				cryptolog.Warningf("key: %s can't be unlocked, err:%s", name, err)
				return err
			}
		} else if strings.HasPrefix(name, Sign.Prefix()) == true {
			privkey, err := ethcrypto.HexToECDSA(val)
			id, err := uuid.NewRandom()
			key := &ethkeystore.Key{
				Id:         id,
				Address:    ethcrypto.PubkeyToAddress(privkey.PublicKey),
				PrivateKey: privkey,
			}

			if err == nil {
				ks.unlocked[name] = key
			} else {
				cryptolog.Warningf("key: %s can't be unlocked, err:%s", name, err)
				return err
			}
		}
	}
	return nil
}

func initConfigfile(dir string, keyname string) (*viper.Viper, error) {
	if err := utils.EnsureDir(dir); err != nil {
		cryptolog.Errorf("mockks directory failed: %s", err)
		return nil, err
	}

	v := viper.New()
	v.SetConfigFile(keyname + "_mockks.toml")
	v.SetConfigName(keyname + "_mockks")
	v.SetConfigType("toml")
	v.AddConfigPath(dir)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			cryptolog.Infof("config file not found, generating...")
			v.Set("mockkeys", map[string]string{})
			v.SafeWriteConfig()
		} else {
			return nil, err
		}
	}
	return v, nil
}

func load(dir string, keyname string) (map[string]string, error) {
	v, err := initConfigfile(dir, keyname)
	if err != nil {
		return nil, err
	}
	err = v.ReadInConfig()
	if err != nil {
		return nil, err
	}

	mockkeys := v.GetStringMapString("mockkeys")
	return mockkeys, nil
}

func (ks *MockKeyStore) writeToconfig() error {
	ks.v.Set("mockkeys", ks.keys)
	return ks.v.WriteConfig()
}

func (ks *MockKeyStore) IfKeyExist(keyname string) (bool, error) {
	_, ok := ks.keys[keyname]
	return ok, nil
}

func (ks *MockKeyStore) GetHexKey(keyname string) (string, error) {
	exist, err := ks.IfKeyExist(keyname)
	if err != nil {
		return "", err
	}
	if exist == true {
		return ks.keys[keyname], nil
	}
	return "", fmt.Errorf("Key '%s' not exists", keyname)

}

func (ks *MockKeyStore) NewKey(keyname string, keytype KeyType, password string) (string, error) {
	//interface{} eth *PublicKey address or *X25519Recipient string, will be upgrade to generics
	ks.mu.Lock()
	defer ks.mu.Unlock()

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
		ks.keys[keyname] = key.String()
		err = ks.writeToconfig()
		if err != nil {
			return "", err
		}
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

		privkeybytes := ethcrypto.FromECDSA(privkey)
		ks.keys[keyname] = hex.EncodeToString(privkeybytes)
		err = ks.writeToconfig()
		key := &ethkeystore.Key{
			Id:         id,
			Address:    ethcrypto.PubkeyToAddress(privkey.PublicKey),
			PrivateKey: privkey,
		}
		ks.unlocked[keyname] = key
		return key.Address.String(), nil
	default:
		return "", fmt.Errorf("unsupported key type")
	}
}

func (ks *MockKeyStore) Lock() error {
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

func (ks *MockKeyStore) Import(keyname string, encodedkey string, keytype KeyType, password string) (string, error) {
	cryptolog.Warningf("======= import key==========")
	keyname = keytype.NameString(keyname)
	switch keytype {
	case Sign:
		privkey, err := ethcrypto.HexToECDSA(encodedkey)
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

		privkeybytes := ethcrypto.FromECDSA(privkey)
		ks.keys[keyname] = hex.EncodeToString(privkeybytes)
		err = ks.writeToconfig()
		address := ethcrypto.PubkeyToAddress(privkey.PublicKey)
		key := &ethkeystore.Key{
			Id:         id,
			Address:    address,
			PrivateKey: privkey,
		}
		ks.unlocked[keyname] = key
		if err == nil {
			cryptolog.Warningf("key %s imported, address: %s", keyname, address)
		}
		return address.String(), err
	}
	return "", nil
}

func (ks *MockKeyStore) Sign(data []byte, privKey p2pcrypto.PrivKey) ([]byte, error) {
	return privKey.Sign(data)
}

func (ks *MockKeyStore) VerifySign(data, sig []byte, pubKey p2pcrypto.PubKey) (bool, error) {
	return pubKey.Verify(data, sig)
}

func (ks *MockKeyStore) GetKeyFromUnlocked(keyname string) (interface{}, error) {
	if val, ok := ks.unlocked[keyname]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("key %s not exist or not be unlocked", keyname)
}

func (ks *MockKeyStore) SignByKeyName(keyname string, data []byte, opts ...string) ([]byte, error) {

	nodeprefix := ""
	if len(opts) == 1 {
		nodeprefix = opts[0] + "_"
	}
	keyname = nodeprefix + keyname

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
}

func (ks *MockKeyStore) VerifySignByKeyName(keyname string, data []byte, sig []byte, opts ...string) (bool, error) {

	nodeprefix := ""
	if len(opts) == 1 {
		nodeprefix = opts[0] + "_"
	}
	keyname = nodeprefix + keyname
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

func (ks *MockKeyStore) EncryptTo(to []string, data []byte) ([]byte, error) {
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

func (ks *MockKeyStore) Decrypt(keyname string, data []byte) ([]byte, error) {
	key, err := ks.GetKeyFromUnlocked(Encrypt.NameString(keyname))
	if err != nil {
		return nil, err
	}
	encryptk, ok := key.(*age.X25519Identity)

	if ok != true {
		return nil, fmt.Errorf("The key %s is not a encrypt key", keyname)
	}
	r, err := age.Decrypt(bytes.NewReader(data), encryptk)
	return ioutil.ReadAll(r)
}

func (ks *MockKeyStore) GetEncodedPubkey(keyname string, keytype KeyType) (string, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

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

func (ks *MockKeyStore) GetPeerInfo(keyname string) (peerid peer.ID, ethaddr string, err error) {
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
