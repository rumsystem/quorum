package crypto

import (
	"fmt"
	"testing"

	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/rumsystem/quorum/internal/pkg/utils"
)

func TestKeyType(t *testing.T) {
	var strList []string
	for i := 0; i < 10; i++ {
		strList = append(strList, utils.GetRandomStr(2*i))
	}

	ktPrefixs := map[KeyType]string{
		Encrypt: "encrypt_",
		Sign:    "sign_",
	}

	for kt, prefix := range ktPrefixs {
		for _, s := range strList {
			if kt.Prefix() != prefix {
				t.Errorf("key type prefix error, expect %s, got %s", prefix, kt.Prefix())
			}

			if kt.NameString(s) != fmt.Sprintf("%s%s", prefix, s) {
				t.Errorf("key type name string error, expect %s, got %s", fmt.Sprintf("%s%s", prefix, s), kt.NameString(s))
			}
		}
	}
}

type FnGetUnlockedKey func(Keystore, string) (interface{}, error)
type FnAliasToKeyname func(Keystore, string) string

func FactoryTestEthSign(ks Keystore, password string, fnGetUnlocked FnGetUnlockedKey) func(t *testing.T) {
	return func(t *testing.T) {
		key1name := "key1"
		_, err := ks.NewKey(key1name, Sign, password)
		keyname := Sign.NameString(key1name)
		key, err := fnGetUnlocked(ks, keyname)
		if err != nil {
			t.Errorf("Get Unlocked key err: %s", err)
		}
		ethkey, ok := key.(*ethkeystore.Key)
		if ok == false {
			t.Errorf("new key is not a eth sign key: %s", key)
		}

		testdata := "some random text for testing"
		testdatahash := Hash([]byte(testdata))
		sig, err := ks.EthSign(testdatahash, ethkey.PrivateKey)
		if err != nil {
			t.Errorf("new key is not a eth sign key: %s", err)
		}
		verifyresult := ks.EthVerifySign(testdatahash, sig, &ethkey.PrivateKey.PublicKey)
		if verifyresult == false {
			t.Errorf("sig verify failure")
		}
	}
}

func FactoryTestAlias(ks Keystore, password string, fnAliasToKeyname FnAliasToKeyname) func(t *testing.T) {
	return func(t *testing.T) {
		var mappingkeyname, mappingkeyname1 string
		//create 6 keyparis, and mapping the fourth keypair to a new keyname
		for i := 0; i < 6; i++ {
			keyname := fmt.Sprintf("key%d", i)
			newsignid, err := ks.NewKey(keyname, Sign, password)
			if err != nil {
				t.Errorf("New sign key err : %s", err)
			}
			//err = nodeoptions.SetSignKeyMap(keyname, newsignid)
			t.Logf("new signkey: %s", newsignid)

			_, err = ks.NewKey(keyname, Encrypt, password)
			if err != nil {
				t.Errorf("New encrypt key err : %s", err)
			}
			if i == 1 {
				mappingkeyname = keyname
			} else if i == 3 {
				mappingkeyname1 = keyname
			}
		}

		t.Log("set keyalias...")
		aliasname := "a_new_mapping_keyname"
		err := ks.NewAlias(aliasname, mappingkeyname, password)
		if err != nil {
			t.Errorf("new keyalias err...%s", err)
		}

		aliasname2 := "a_new_mapping_keyname_2"
		err = ks.NewAlias(aliasname2, mappingkeyname, password)
		if err != nil {
			t.Errorf("new keyalias2 err...%s", err)
		}

		aliasname3 := "a_new_mapping_keyname_3"
		err = ks.NewAlias(aliasname3, mappingkeyname1, password)
		if err != nil {
			t.Errorf("new keyalias3 err...%s", err)
		}

		keyname := fnAliasToKeyname(ks, aliasname)
		if keyname == mappingkeyname {
			t.Logf("alias %s is keyname %s", aliasname, keyname)
		} else {
			t.Errorf("get alias %s err, can't find this alias", keyname)
		}

		err = ks.NewAlias(aliasname, mappingkeyname, password)
		if err == nil {
			t.Errorf("repeat new keyalias should be failed")
		}

		t.Log("try unalias...")
		err = ks.UnAlias(aliasname, password)
		if err != nil {
			t.Errorf("UnAlias err: %s", err)
		}
		t.Log("OK")
		t.Log("try unalias again...")

		err = ks.UnAlias(aliasname, password)
		if err == nil {
			t.Errorf("repeat unalias should be failed")
		}
		t.Log("OK")

		t.Log("try unalias not exist alias...")
		err = ks.UnAlias("not_exist_alias", password)
		if err == nil {
			t.Errorf("unalias not exist alias should be failed")
		}
		t.Log("OK")

		t.Log("try get encoded pubkey by alias...")
		pubkeybyalias, getkeyerr := ks.GetEncodedPubkeyByAlias(aliasname3, Sign)
		if getkeyerr != nil {
			t.Errorf("GetEncodedPubkeyByAlias with alias %s err: %s", aliasname3, getkeyerr)
		}

		pubkeybyname, getkeyerr := ks.GetEncodedPubkey(mappingkeyname1, Sign)
		if getkeyerr != nil {
			t.Errorf("GetEncodedPubkeyByAlias with name %s err: %s", pubkeybyname, getkeyerr)
		}

		_, getencryptkeyerr := ks.GetEncodedPubkey(mappingkeyname1, Encrypt)
		if getencryptkeyerr != nil {
			t.Errorf("GetEncodedPubkeyByAlias Encrypt with name %s err: %s", pubkeybyname, getkeyerr)
		}

		if pubkeybyalias != pubkeybyname {
			t.Errorf("GetEncodedPubkey ByAlias or ByName should be equal.")
		}
		aliaslist := ks.GetAlias(mappingkeyname1)
		if len(aliaslist) != 1 {
			t.Errorf("GetAlias of %s err", mappingkeyname1)
		}

		t.Log("try sign by alias...")
		testdata := "some random text for testing"
		testdatahash := Hash([]byte(testdata))
		signbyaliasresult, signerr := ks.SignByKeyAlias(aliasname3, testdatahash)
		if signerr != nil {
			t.Errorf("SignByKeyAlias with alias %s err: %s", aliasname3, signerr)
		}

		verifyresult, verifyerr := ks.VerifySignByKeyName(mappingkeyname1, testdatahash, signbyaliasresult)
		if verifyresult == false {
			t.Errorf("SignByKeyAlias %s verify err: %s", aliasname3, verifyerr)
		}

		verifyresult, _ = ks.VerifySignByKeyName(mappingkeyname, testdatahash, signbyaliasresult)
		if verifyresult == true {
			t.Errorf("SignByKeyAlias %s verify by %s should be failed", aliasname3, mappingkeyname)
		}
		t.Log("OK")
	}
}

func FactoryTestNewEncryptKey(ks Keystore, password string) func(*testing.T) {
	return func(t *testing.T) {
		key1name := "key1"
		newencryptid, err := ks.NewKey(key1name, Encrypt, password)
		if err != nil {
			t.Errorf("New encrypt key err : %s", err)
		}
		data := "secret message"
		encryptdata, err := ks.EncryptTo([]string{newencryptid}, []byte(data))
		if err != nil {
			t.Errorf("encrypt data error : %s", err)
		}

		decrypteddata, err := ks.Decrypt(key1name, encryptdata)
		if err != nil {
			t.Errorf("decrypt data error : %s", err)
		}
		if string(decrypteddata) != data {
			t.Errorf("decrypt data is not matched with orginal: %s / %s",
				string(decrypteddata), data)
		}
	}
}

func FactoryTestImportSignKey(ks Keystore) func(*testing.T) {
	return func(t *testing.T) {
		key1name := "importedkey1"
		key1addr := "0x57c8CBB7966AAC85b32cB6827C0c14A4ae4Af0CE"
		address, err := ks.Import(key1name, "84f8da8f95760fa3d0b6632ef66b89ea255a85974eccad7642ef12c4265677e0", Sign, "a.Passw0rda")
		if err != nil {
			t.Errorf("Get Unlocked key err: %s", err)
		}
		if address != key1addr {
			t.Errorf("key import is not matched: %s / %s ", address, key1addr)
		}
	}
}

func FactoryTestNewSignKey(ks Keystore, password string, fnGetUnlocked FnGetUnlockedKey) func(*testing.T) {
	return func(t *testing.T) {
		key1name := "key1"
		newsignaddr, err := ks.NewKey(key1name, Sign, password)
		keyname := Sign.NameString(key1name)
		key, err := fnGetUnlocked(ks, keyname)
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

		signature, err := ks.SignByKeyName(key1name, []byte("a test string"))
		if err != nil {
			t.Errorf("Signnature err: %s", err)
		}

		//should succ
		result, err := ks.VerifySignByKeyName(key1name, []byte("a test string"), signature)
		if err != nil {
			t.Errorf("Verify signnature err: %s", err)
		}

		if result == false {
			t.Errorf("signnature verify should successded but failed.")
		}

		//should fail
		result, err = ks.VerifySignByKeyName(key1name, []byte("a new string"), signature)
		if err != nil {
			t.Errorf("Verify signnature err: %s", err)
		}

		if result == true {
			t.Errorf("signnature verify should failed, but it succeeded.")
		}
	}
}
