//go:build js && wasm
// +build js,wasm

package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"syscall/js"

	"filippo.io/age"
	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/rumsystem/keystore/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/appdata"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumStorage "github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

var backupLogger = logging.Logger("backup")

func getKeystore(password string) ([]string, error) {
	ret := []string{}

	idb := quorumStorage.QSIndexDB{}
	err := idb.Init("keystore")
	if err != nil {
		return nil, err
	}

	r, err := age.NewScryptRecipient(password)
	if err != nil {
		return nil, err
	}

	idb.Foreach(func(k, v []byte, e error) error {
		if e != nil {
			return e
		}

		pair := make(map[string]interface{})
		key := string(k)
		pair["key"] = key
		pair["value"] = base64.StdEncoding.EncodeToString(v)

		if strings.HasPrefix(key, crypto.Sign.Prefix()) {
			key, err := ethkeystore.DecryptKey(v, password)
			if err != nil {
				backupLogger.Fatalf(err.Error())
			}
			privKey := key.PrivateKey
			addr := ethcrypto.PubkeyToAddress(privKey.PublicKey)
			// Make sure we're really operating on the requested key (no swap attacks)
			if key.Address != addr {
				backupLogger.Fatalf("key content mismatch: have account %x, want %x", key.Address, addr)
			}
			pair["addr"] = addr
		}

		backupLogger.Info("exporting " + key)

		kvBytes, err := json.Marshal(pair)
		if err != nil {
			return err
		}

		output := new(bytes.Buffer)
		if err := crypto.AgeEncrypt([]age.Recipient{r}, bytes.NewReader(kvBytes), output); err != nil {
			return err
		}
		encryptedKvBytes, err := ioutil.ReadAll(output)
		if err != nil {
			return err
		}
		ret = append(ret, base64.StdEncoding.EncodeToString(encryptedKvBytes))
		return nil
	})
	return ret, nil
}

func getSeeds(password string) ([]handlers.GroupSeed, error) {
	idb := quorumStorage.QSIndexDB{}
	err := idb.Init("app")
	if err != nil {
		return nil, err
	}
	appDb := appdata.NewAppDb()
	appDb.Db = &idb

	return handlers.GetAllGroupSeeds(appDb)
}

func BackupWasmRaw(password string, onWrite func(string), onFinish func()) error {
	// keystore
	ks, err := getKeystore(password)
	if err != nil {
		return err
	}
	backupObj := handlers.QuorumWasmExportObject{}
	backupObj.Keystore = ks

	// seeds
	seeds, err := getSeeds(password)
	backupObj.Seeds = seeds

	// flush out
	res, err := json.Marshal(backupObj)
	onWrite(string(res))
	onFinish()
	return nil
}

// password should be the keystore password, it is used to get the addr
func KeystoreBackupRaw(password string, onWrite func(string), onFinish func()) error {
	idb := quorumStorage.QSIndexDB{}
	err := idb.Init("keystore")
	if err != nil {
		return err
	}

	r, err := age.NewScryptRecipient(password)
	if err != nil {
		return err
	}

	idb.Foreach(func(k, v []byte, e error) error {
		if e != nil {
			return e
		}

		pair := make(map[string]interface{})
		key := string(k)
		pair["key"] = key
		pair["value"] = base64.StdEncoding.EncodeToString(v)

		if strings.HasPrefix(key, crypto.Sign.Prefix()) {
			key, err := ethkeystore.DecryptKey(v, password)
			if err != nil {
				backupLogger.Fatalf(err.Error())
			}
			privKey := key.PrivateKey
			addr := ethcrypto.PubkeyToAddress(privKey.PublicKey)
			// Make sure we're really operating on the requested key (no swap attacks)
			if key.Address != addr {
				backupLogger.Fatalf("key content mismatch: have account %x, want %x", key.Address, addr)
			}
			pair["addr"] = addr
		}

		backupLogger.Info("exporting " + key)

		kvBytes, err := json.Marshal(pair)
		if err != nil {
			return err
		}

		output := new(bytes.Buffer)
		if err := crypto.AgeEncrypt([]age.Recipient{r}, bytes.NewReader(kvBytes), output); err != nil {
			return err
		}
		encryptedKvBytes, err := ioutil.ReadAll(output)
		if err != nil {
			return err
		}
		res := base64.StdEncoding.EncodeToString(encryptedKvBytes)
		onWrite(res)
		return nil
	})
	onFinish()
	return nil
}

func GetFnReadableStream(password string, fn func(string, func(string), func()) error) js.Value {
	underlyingSource := map[string]interface{}{
		"start": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			controller := args[0]
			go func() {
				fn(
					password,
					func(str string) {
						controller.Call("enqueue", js.ValueOf(str+"\n"))
					},
					func() {
						controller.Call("close")
					},
				)
			}()
			return nil
		}),
	}

	return js.Global().Get("ReadableStream").New(underlyingSource)
}

func GetKeystoreBackupReadableStream(password string) js.Value {
	return GetFnReadableStream(password, KeystoreBackupRaw)
}

func GetBackupReadableStream(password string) js.Value {
	return GetFnReadableStream(password, BackupWasmRaw)
}

func KeystoreRestoreRaw(password string, keystoreStr string) error {
	idb := quorumStorage.QSIndexDB{}
	err := idb.Init("keystore")
	if err != nil {
		return err
	}

	identities := []age.Identity{
		&crypto.LazyScryptIdentity{password},
	}

	for _, row := range strings.Split(keystoreStr, "\n") {
		enc, err := base64.StdEncoding.DecodeString(row)
		if err != nil {
			return fmt.Errorf("base64 decode config data failed: %s", err)
		}

		r, err := age.Decrypt(bytes.NewReader(enc), identities...)
		if err != nil {
			return fmt.Errorf("decrypt config data failed: %v", err)
		}

		kvBytes, err := ioutil.ReadAll(r)
		if err != nil {
			return fmt.Errorf("ioutil.ReadAll config failed: %v", err)
		}
		pair := make(map[string]interface{})
		err = json.Unmarshal(kvBytes, &pair)
		if err != nil {
			return err
		}
		k := pair["key"].(string)
		v, _ := base64.StdEncoding.DecodeString(pair["value"].(string))
		backupLogger.Info("Loading " + k)

		err = idb.Set([]byte(k), v)
		if err != nil {
			return err
		}
		backupLogger.Info("OK")
	}

	return nil
}

func RestoreWasmRaw(password string, backupContent string) error {
	idb := quorumStorage.QSIndexDB{}
	err := idb.Init("keystore")
	if err != nil {
		return err
	}

	identities := []age.Identity{
		&crypto.LazyScryptIdentity{password},
	}

	backupObj := handlers.QuorumWasmExportObject{}
	err = json.Unmarshal([]byte(backupContent), &backupObj)
	if err != nil {
		return err
	}

	for _, ks := range backupObj.Keystore {
		enc, err := base64.StdEncoding.DecodeString(ks)
		if err != nil {
			return fmt.Errorf("base64 decode config data failed: %s", err)
		}

		r, err := age.Decrypt(bytes.NewReader(enc), identities...)
		if err != nil {
			return fmt.Errorf("decrypt config data failed: %v", err)
		}

		kvBytes, err := ioutil.ReadAll(r)
		if err != nil {
			return fmt.Errorf("ioutil.ReadAll config failed: %v", err)
		}
		pair := make(map[string]interface{})
		err = json.Unmarshal(kvBytes, &pair)
		if err != nil {
			return err
		}
		k := pair["key"].(string)
		v, _ := base64.StdEncoding.DecodeString(pair["value"].(string))
		backupLogger.Info("Loading " + k)

		err = idb.Set([]byte(k), v)
		if err != nil {
			return err
		}
		backupLogger.Info("OK")
	}

	// seeds are ignored by now, could done by js side

	return nil
}
