//go:build !js
// +build !js

package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"filippo.io/age"
	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	p2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	peer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/viper"
)

type DirKeyStore struct {
	Name         string
	KeystorePath string
	password     string
	unlocked     map[string]interface{} //eth *Key or *X25519Identity, will be upgrade to generics
	signkeymap   map[string]string
	keyaliasmap  map[string]string
	unlockTime   time.Time
	v            *viper.Viper
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
	v, keyaliasmap, err := loadAliasmap(keydir)
	if err != nil {
		return nil, 0, err
	}

	ks := &DirKeyStore{Name: name, KeystorePath: keydir, unlocked: make(map[string]interface{}), keyaliasmap: keyaliasmap, signkeymap: make(map[string]string), v: v}
	return ks, signkeycount, nil
}

func loadAliasmap(dir string) (*viper.Viper, map[string]string, error) {
	v, err := initConfigfile(dir)
	err = v.ReadInConfig()
	if err != nil {
		return nil, nil, err
	}

	return v, v.GetStringMapString("aliaskeymap"), nil
}

func initConfigfile(dir string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigFile("alias.toml")
	v.SetConfigName("alias")
	v.SetConfigType("toml")
	v.AddConfigPath(dir)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			writeDefaultToconfig(v)
		} else {
			return nil, err
		}
	}

	return v, nil
}

func writeDefaultToconfig(v *viper.Viper) error {
	v.Set("AliasKeyMap", map[string]string{})
	return v.SafeWriteConfig()
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

func (ks *DirKeyStore) DeleteKeyfile(filename string) error {
	storefilename := JoinKeyStorePath(ks.KeystorePath, filename)
	return os.Remove(storefilename)
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

func (ks *DirKeyStore) NewAlias(keyalias, keyname, password string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	err := ks.CanAliasKey(keyalias, keyname, password)
	if err == nil {
		ks.keyaliasmap[keyalias] = keyname
		writeToconfig(ks.v, ks.keyaliasmap)
		return nil
	}
	return err
}

func (ks *DirKeyStore) UnAlias(keyalias, password string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	err := ks.CanUnAliasKey(keyalias, password)
	if err == nil { //ok can alias
		delete(ks.keyaliasmap, keyalias)
		return writeToconfig(ks.v, ks.keyaliasmap)
	} else {
		return err
	}
}

// check the keyname of an alias, return keyname
func (ks *DirKeyStore) AliasToKeyname(keyalias string) string {
	k, ok := ks.keyaliasmap[keyalias]
	if ok == true {
		return k
	}
	return ""
}

func writeToconfig(v *viper.Viper, keyaliasmap map[string]string) error {
	v.Set("AliasKeyMap", keyaliasmap)
	return v.WriteConfig()
}

func (ks *DirKeyStore) CanAliasKey(keyalias string, keyname string, password string) error {
	//TODO: more verify
	if ks.AliasToKeyname(keyalias) == "" {
		return nil //ok can mapping
	}
	return errors.New("alias exists")
}

func (ks *DirKeyStore) CanUnAliasKey(keyalias string, password string) error {
	//TODO: more verify
	if ks.AliasToKeyname(keyalias) == "" {
		return errors.New("alias not find")
	}
	return nil
}

func (ks *DirKeyStore) Sign(data []byte, privKey p2pcrypto.PrivKey) ([]byte, error) {
	return privKey.Sign(data)
}

func (ks *DirKeyStore) EthSign(digestHash []byte, privKey *ecdsa.PrivateKey) ([]byte, error) {
	return ethcrypto.Sign(digestHash, privKey)
}

func (ks *DirKeyStore) SignByKeyAlias(keyalias string, data []byte, opts ...string) ([]byte, error) {
	keyname := ks.AliasToKeyname(keyalias)
	if keyname == "" {
		//alias not exist
		return nil, fmt.Errorf("The key alias %s is not exist", keyalias)
	} else {
		return ks.SignByKeyName(keyname, data, opts...)
	}
}

func (ks *DirKeyStore) EthSignByKeyAlias(keyalias string, digestHash []byte, opts ...string) ([]byte, error) {
	keyname := ks.AliasToKeyname(keyalias)
	if keyname == "" {
		//alias not exist
		return nil, fmt.Errorf("The key alias %s is not exist", keyalias)
	} else {
		return ks.EthSignByKeyName(keyname, digestHash, opts...)
	}
}

func (ks *DirKeyStore) EthSignByKeyName(keyname string, digestHash []byte, opts ...string) ([]byte, error) {
	key, err := ks.GetKeyFromUnlocked(Sign.NameString(keyname))
	if err != nil {
		return nil, err
	}
	signk, ok := key.(*ethkeystore.Key)
	if ok != true {
		return nil, fmt.Errorf("The key %s is not a Sign key", keyname)
	}
	return ethcrypto.Sign(digestHash, signk.PrivateKey)
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
}

// SignTxByKeyName sign tx with keyname
func (ks *DirKeyStore) SignTxByKeyName(keyname string, nonce uint64, to common.Address, value *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, chainID *big.Int) (string, error) {
	key, err := ks.GetKeyFromUnlocked(Sign.NameString(keyname))
	if err != nil {
		return "", err
	}
	signk, ok := key.(*ethkeystore.Key)
	if !ok {
		return "", fmt.Errorf("The key %s is not a Sign key", keyname)
	}

	tx := types.NewTransaction(nonce, to, value, gasLimit, gasPrice, data)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), signk.PrivateKey)
	if err != nil {
		return "", err
	}

	signedTxData, err := signedTx.MarshalBinary()
	if err != nil {
		return "", err
	}
	return hexutil.Encode(signedTxData), nil
}

// SignTxByKeyAlias sign tx with key alias
func (ks *DirKeyStore) SignTxByKeyAlias(keyalias string, nonce uint64, to common.Address, value *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, chainID *big.Int) (string, error) {
	keyname := ks.AliasToKeyname(keyalias)
	if keyname == "" {
		return "", fmt.Errorf("The key alias %s is not exist", keyalias)
	}

	return ks.SignTxByKeyName(keyname, nonce, to, value, gasLimit, gasPrice, data, chainID)
}

func (ks *DirKeyStore) VerifySign(data, sig []byte, pubKey p2pcrypto.PubKey) (bool, error) {
	return pubKey.Verify(data, sig)
}

func (ks *DirKeyStore) EthVerifyByKeyName(keyname string, digestHash, signature []byte) (bool, error) {
	key, err := ks.GetKeyFromUnlocked(Sign.NameString(keyname))
	if err != nil {
		return false, err
	}
	signk, ok := key.(*ethkeystore.Key)
	if ok != true {
		return false, fmt.Errorf("The key %s is not a Sign key", keyname)
	}
	publicKey := signk.PrivateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return false, fmt.Errorf("error casting public key to ECDSA")
	}

	verified := ks.EthVerifySign(digestHash, signature, publicKeyECDSA)
	return verified, nil
}

func (ks *DirKeyStore) EthVerifySign(digestHash, signature []byte, pubKey *ecdsa.PublicKey) bool {
	sig := signature[:len(signature)-1] // remove recovery id
	return ethcrypto.VerifySignature(ethcrypto.FromECDSAPub(pubKey), digestHash, sig)
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
			return base64.RawURLEncoding.EncodeToString(ethcrypto.CompressPubkey(&signk.PrivateKey.PublicKey)), nil
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

func (ks *DirKeyStore) GetEncodedPubkeyByAlias(keyalias string, keytype KeyType) (string, error) {
	keyname := ks.AliasToKeyname(keyalias)
	if keyname == "" {
		//alias not exist
		return "", fmt.Errorf("The key alias %s is not exist", keyalias)
	} else {
		return ks.GetEncodedPubkey(keyname, keytype)
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

func (ks *DirKeyStore) DecryptByAlias(keyalias string, data []byte) ([]byte, error) {
	keyname := ks.AliasToKeyname(keyalias)
	if keyname == "" {
		//alias not exist
		return nil, fmt.Errorf("The key alias %s is not exist", keyalias)
	} else {
		return ks.Decrypt(keyname, data)
	}

}

func (ks *DirKeyStore) RemoveKey(keyname string, keytype KeyType) (err error) {
	keyname = keytype.NameString(keyname)
	exist, err := ks.IfKeyExist(keyname)
	if err != nil {
		return err
	}
	if exist != true {
		return fmt.Errorf("Key '%s' not exists", keyname)
	}
	ks.DeleteKeyfile(keyname)

	return nil
}

func (ks *DirKeyStore) GetAlias(keyname string) []string {
	aliaslist := []string{}
	for a, k := range ks.keyaliasmap {
		if k == keyname {
			aliaslist = append(aliaslist, a)
		}
	}
	return aliaslist
}

func (ks *DirKeyStore) ListAll() (keys []*KeyItem, err error) {
	files, err := ioutil.ReadDir(ks.KeystorePath)
	if err != nil {
		return nil, err
	}
	items := []*KeyItem{}
	for _, f := range files {
		file := f.Name()
		if strings.HasPrefix(file, Sign.Prefix()) == true {
			name := file[len(Sign.Prefix()):]
			alias := ks.GetAlias(name)
			item := &KeyItem{Keyname: name, Alias: alias, Type: Sign}
			items = append(items, item)
		} else if strings.HasPrefix(file, Encrypt.Prefix()) == true {
			name := file[len(Encrypt.Prefix()):]
			alias := ks.GetAlias(name)
			item := &KeyItem{Keyname: name, Alias: alias, Type: Encrypt}
			items = append(items, item)
		}
	}
	return items, nil
}
