package crypto

import (
	"fmt"
	"log"
	"testing"

	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestInitDirKeyStore(t *testing.T) {
	name := "testkeystore"
	tempdir := fmt.Sprintf("%s/%s", t.TempDir(), name)
	log.Printf("tempdir %s", tempdir)
	_, count, err := InitDirKeyStore(name, tempdir)
	if err != nil {
		t.Errorf("keystore init err: %s", err)
	}
	if count != 0 {
		t.Errorf("init new keystore count should be 0, not : %d", count)
	}
}

func TestNewSignKey(t *testing.T) {
	name := "testnewkey"

	password := "my.Passw0rd"
	tempdir := fmt.Sprintf("%s/%s", t.TempDir(), name)
	dirks, _, err := InitDirKeyStore(name, tempdir)
	if err != nil {
		t.Errorf("keystore init err: %s", err)
	}
	key1name := "key1"
	newsignaddr, err := dirks.NewKey(key1name, Sign, password)
	keyname := Sign.NameString(key1name)
	key, err := dirks.GetKeyFromUnlocked(keyname)
	if err != nil {
		t.Errorf("Get Unlocked key err: %s", err)
	}
	ethkey, ok := key.(*ethkeystore.Key)
	if ok == false {
		t.Errorf("new key is not a eth sign key: %s", key)
	}
	pubaddress := ethcrypto.PubkeyToAddress(ethkey.PrivateKey.PublicKey).Hex()
	if pubaddress != newsignaddr {
		t.Errorf("new key address is not matched %s / %s", pubaddress, newsignaddr)
	}

	signature, err := dirks.SignByKeyName(key1name, []byte("a test string"))
	if err != nil {
		t.Errorf("Signnature err: %s", err)
	}

	//should succ
	result, err := dirks.VerifySignByKeyName(key1name, []byte("a test string"), signature)
	if err != nil {
		t.Errorf("Verify signnature err: %s", err)
	}

	if result == false {
		t.Errorf("signnature verify should successded but failed.")
	}

	//should fail
	result, err = dirks.VerifySignByKeyName(key1name, []byte("a new string"), signature)
	if err != nil {
		t.Errorf("Verify signnature err: %s", err)
	}

	if result == true {
		t.Errorf("signnature verify should failed, but it succeeded.")
	}
}

func TestImportSignKey(t *testing.T) {
	name := "testnewkey"
	tempdir := fmt.Sprintf("%s/%s", t.TempDir(), name)
	log.Printf("tempdir %s", tempdir)
	dirks, _, err := InitDirKeyStore(name, tempdir)
	if err != nil {
		t.Errorf("keystore init err: %s", err)
	}
	key1name := "key1"
	key1addr := "0x57c8CBB7966AAC85b32cB6827C0c14A4ae4Af0CE"
	address, err := dirks.Import(key1name, "84f8da8f95760fa3d0b6632ef66b89ea255a85974eccad7642ef12c4265677e0", Sign, "a.Passw0rda")
	if err != nil {
		t.Errorf("Get Unlocked key err: %s", err)
	}
	if address != key1addr {
		t.Errorf("key import is not matched: %s / %s ", address, key1addr)
	}
}

func TestNewEncryptKey(t *testing.T) {
	name := "testnewkey"
	password := "my.Passw0rd"
	tempdir := fmt.Sprintf("%s/%s", t.TempDir(), name)
	dirks, _, err := InitDirKeyStore(name, tempdir)
	if err != nil {
		t.Errorf("keystore init err: %s", err)
	}
	key1name := "key1"
	newencryptid, err := dirks.NewKey(key1name, Encrypt, password)
	if err != nil {
		t.Errorf("New encrypt key err : %s", err)
	}
	data := "secret message"
	encryptdata, err := dirks.EncryptTo([]string{newencryptid}, []byte(data))
	if err != nil {
		t.Errorf("encrypt data error : %s", err)
	}

	decrypteddata, err := dirks.Decrypt(key1name, encryptdata)
	if err != nil {
		t.Errorf("decrypt data error : %s", err)
	}
	if string(decrypteddata) != data {
		t.Errorf("decrypt data is not matched with orginal: %s / %s", string(decrypteddata), data)
	}
}
