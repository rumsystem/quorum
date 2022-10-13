//go:build js && wasm
// +build js,wasm

package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"filippo.io/age"
	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	p2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	peer "github.com/libp2p/go-libp2p/core/peer"
	quorumStorage "github.com/rumsystem/quorum/pkg/storage"
)

type BrowserKeystore struct {
	store    *quorumStorage.QSIndexDB
	cache    map[string]interface{}
	password string
}

func InitBrowserKeystore(password string) (Keystore, error) {
	bks := BrowserKeystore{}
	bks.cache = make(map[string]interface{})
	db := quorumStorage.QSIndexDB{}
	err := db.Init("keystore")
	if err != nil {
		return nil, err
	}

	bks.store = &db
	bks.password = password
	ks = &bks

	_, err = bks.store.Count()
	if err != nil {
		return nil, err
	}

	defaultKeyName := "default"
	k, err := bks.GetUnlockedKey(Sign.NameString(defaultKeyName))
	if k == nil && strings.HasPrefix(err.Error(), "key not exist") {
		// init default signkey
		_, err := ks.NewKey(defaultKeyName, Sign, password)
		if err != nil {
			return nil, err
		}
	}

	return &bks, nil
}

func (ks *BrowserKeystore) Unlock(signkeymap map[string]string, password string) error {
	ks.password = password
	return nil
}

func (ks *BrowserKeystore) Backup([]byte) (string, string, string, error) {
	// TODO
	return "", "", "", errors.New("Not Implement Yet")
}
func (ks *BrowserKeystore) Restore(encGroupSeed, encKeystore, encConfig, path, password string) error {
	// TODO
	return errors.New("Not Implement Yet")
}

func (ks *BrowserKeystore) Lock() error {
	for k, _ := range ks.cache {
		if strings.HasPrefix(k, Sign.Prefix()) { //zero the signkey in the memory
			signk, ok := ks.cache[k].(*ethkeystore.Key)
			if ok != true {
				return fmt.Errorf("The key %s is not a Sign key", k)
			}
			zeroSignKey(signk.PrivateKey)
			ks.cache[k] = nil
		}
		if strings.HasPrefix(k, Encrypt.Prefix()) {
			var zero = &age.X25519Identity{}
			ks.cache[k] = zero
		}
	}
	ks.cache = make(map[string]interface{})
	return nil
}

func (ks *BrowserKeystore) NewKey(keyname string, keytype KeyType, password string) (string, error) {
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

func (ks *BrowserKeystore) NewKeyWithDefaultPassword(keyname string, keytype KeyType) (string, error) {
	return ks.NewKey(keyname, keytype, ks.password)
}

func (ks *BrowserKeystore) Import(keyname string, encodedkey string, keytype KeyType, _ string) (string, error) {
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

func (ks *BrowserKeystore) Sign(data []byte, privKey p2pcrypto.PrivKey) ([]byte, error) {
	return privKey.Sign(data)
}

func (ks *BrowserKeystore) SignByKeyAlias(alias string, data []byte, opts ...string) ([]byte, error) {
	keyname, err := ks.GetKeynameFromAlias(alias)
	if err != nil {
		return nil, err
	}
	return ks.SignByKeyName(keyname, data, opts...)
}

func (ks *BrowserKeystore) VerifySign(data, sig []byte, pubKey p2pcrypto.PubKey) (bool, error) {
	return pubKey.Verify(data, sig)
}

func (ks *BrowserKeystore) EthVerifySign(data, signature []byte, pubKey *ecdsa.PublicKey) bool {
	sig := signature[:len(signature)-1] // remove recovery id
	return ethcrypto.VerifySignature(ethcrypto.FromECDSAPub(pubKey), data, sig)
}

func (ks *BrowserKeystore) SignByKeyName(keyname string, data []byte, opts ...string) ([]byte, error) {
	key, err := ks.GetUnlockedKey(Sign.NameString(keyname))
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

func (ks *BrowserKeystore) VerifySignByKeyName(keyname string, data []byte, sig []byte, opts ...string) (bool, error) {
	key, err := ks.GetUnlockedKey(Sign.NameString(keyname))
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

func (ks *BrowserKeystore) EncryptTo(to []string, data []byte) ([]byte, error) {
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

func (ks *BrowserKeystore) Decrypt(keyname string, data []byte) ([]byte, error) {
	key, err := ks.GetUnlockedKey(Encrypt.NameString(keyname))
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

func (ks *BrowserKeystore) GetEncodedPubkey(keyname string, keytype KeyType) (string, error) {
	key, err := ks.GetUnlockedKey(keytype.NameString(keyname))
	if err != nil {
		return "", err
	}
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
}

func (ks *BrowserKeystore) GetPeerInfo(keyname string) (peerid peer.ID, ethaddr string, err error) {
	key, err := ks.GetUnlockedKey(Sign.NameString(keyname))
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

// migrate to eth version
func (ks *BrowserKeystore) EthSign(data []byte, privKey *ecdsa.PrivateKey) ([]byte, error) {
	return ethcrypto.Sign(data, privKey)
}

func (ks *BrowserKeystore) EthSignByKeyAlias(alias string, data []byte, opts ...string) ([]byte, error) {
	keyname, err := ks.GetKeynameFromAlias(alias)
	if err != nil {
		return nil, err
	}
	return ks.EthSignByKeyName(keyname, data, opts...)
}

func (ks *BrowserKeystore) EthSignByKeyName(keyname string, data []byte, opts ...string) ([]byte, error) {
	key, err := ks.GetUnlockedKey(Sign.NameString(keyname))
	if err != nil {
		return nil, err
	}
	signk, ok := key.(*ethkeystore.Key)
	if ok != true {
		return nil, fmt.Errorf("The key %s is not a Sign key", keyname)
	}
	return ethcrypto.Sign(data, signk.PrivateKey)
}

// alias
const ALIAS_PREFIX = "ALIAS"

func getAliasKeyName(alias string) string {
	return fmt.Sprintf("%s_%s", ALIAS_PREFIX, alias)
}

func getAliasFromKey(aliasKey string) string {
	return strings.ReplaceAll(aliasKey, fmt.Sprintf("%s_", ALIAS_PREFIX), "")
}

func (ks *BrowserKeystore) GetKeynameFromAlias(alias string) (string, error) {
	v, err := ks.store.Get([]byte(getAliasKeyName(alias)))
	if err != nil {
		return "", err
	}
	return string(v), nil
}

func (ks *BrowserKeystore) NewAlias(alias, keyname, password string) error {
	// password not used yet
	aliasKey := getAliasKeyName(alias)
	isExist, err := ks.store.IsExist([]byte(aliasKey))
	if err != nil {
		return err
	}
	if isExist {
		return errors.New("alias exists")
	}
	return ks.store.Set([]byte(aliasKey), []byte(keyname))
}

func (ks *BrowserKeystore) UnAlias(alias, password string) error {
	aliasKey := getAliasKeyName(alias)
	isExist, err := ks.store.IsExist([]byte(aliasKey))
	if err != nil {
		return err
	}
	if !isExist {
		return errors.New("alias not exists")
	}
	return ks.store.Delete([]byte(aliasKey))
}

func (ks *BrowserKeystore) GetAlias(keyname string) []string {
	res := []string{}
	ks.store.PrefixForeach([]byte(ALIAS_PREFIX), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		if string(v) == keyname {
			res = append(res, getAliasFromKey(string(k)))
		}
		return nil
	})
	return res
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

	// Skip address validate for browser, it's very slow
	return ks.store.Set([]byte(k), enc)
}

/* this operation is very slow in browser(ethkeystore.DecryptKey) */
func (ks *BrowserKeystore) GetUnlockedKey(keyname string) (interface{}, error) {
	/* check cache first */
	data, ok := ks.cache[keyname]
	if ok && data != nil {
		return data, nil
	}

	/* not in cache, we find it in the encrypted store */
	exist, _ := ks.store.IsExist([]byte(keyname))
	if !exist {
		return nil, fmt.Errorf("key not exist :%s", keyname)
	}

	keyBytes, err := ks.store.Get([]byte(keyname))
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(keyname, Sign.Prefix()) {
		key, err := ethkeystore.DecryptKey(keyBytes, ks.password)
		if err != nil {
			return nil, err
		}
		privKey := key.PrivateKey
		addr := ethcrypto.PubkeyToAddress(privKey.PublicKey)
		// Make sure we're really operating on the requested key (no swap attacks)
		if key.Address != addr {
			return nil, fmt.Errorf("key content mismatch: have account %x, want %x", key.Address, addr)
		}
		ks.cache[keyname] = key
		return key, nil
	} else if strings.HasPrefix(keyname, Encrypt.Prefix()) {
		key, err := AgeDecryptIdentityWithPassword(bytes.NewReader(keyBytes), nil, ks.password)
		if err != nil {
			ks.cache[keyname] = key
		}
		return key, err
	}

	return nil, fmt.Errorf("key %s not exist or not be unlocked", keyname)
}

func (ks *BrowserKeystore) RemoveKey(keyname string, keytype KeyType) (err error) {
	return ks.store.Delete([]byte(keyname))
}

func (ks *BrowserKeystore) ListAll() (keys []*KeyItem, err error) {
	items := []*KeyItem{}
	ks.store.Foreach(func(k []byte, v []byte, err error) error {
		key := string(k)
		if strings.HasPrefix(key, ALIAS_PREFIX) {
			return nil
		}
		if strings.HasPrefix(key, Sign.Prefix()) {
			name := key[len(Sign.Prefix()):]
			alias := ks.GetAlias(name)
			item := &KeyItem{Keyname: name, Alias: alias, Type: Sign}
			items = append(items, item)
		} else if strings.HasPrefix(key, Encrypt.Prefix()) {
			name := key[len(Encrypt.Prefix()):]
			alias := ks.GetAlias(name)
			item := &KeyItem{Keyname: name, Alias: alias, Type: Encrypt}
			items = append(items, item)
		}

		return nil
	})
	return items, nil
}

func (ks *BrowserKeystore) DecryptByAlias(alias string, data []byte) ([]byte, error) {
	keyname, err := ks.GetKeynameFromAlias(alias)
	if err != nil {
		return nil, err
	}
	return ks.Decrypt(keyname, data)
}

func (ks *BrowserKeystore) GetEncodedPubkeyByAlias(alias string, keytype KeyType) (string, error) {
	keyname, err := ks.GetKeynameFromAlias(alias)
	if err != nil {
		return "", err
	}
	return ks.GetEncodedPubkey(keyname, keytype)
}

// SignTxByKeyName sign tx with keyname
func (ks *BrowserKeystore) SignTxByKeyName(keyname string, nonce uint64, to common.Address, value *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, chainID *big.Int) (string, error) {
	key, err := ks.GetUnlockedKey(Sign.NameString(keyname))
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

func (ks *BrowserKeystore) SignTxByKeyAlias(alias string, nonce uint64, to common.Address, value *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, chainID *big.Int) (string, error) {
	keyname, err := ks.GetKeynameFromAlias(alias)
	if err != nil {
		return "", err
	}
	return ks.SignTxByKeyName(keyname, nonce, to, value, gasLimit, gasPrice, data, chainID)
}
