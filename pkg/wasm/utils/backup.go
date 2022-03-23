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
	"github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumStorage "github.com/rumsystem/quorum/internal/pkg/storage"
)

var backupLogger = logging.Logger("backup")

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
		pair["value"] = string(v)
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

func GetKeystoreBackupReadableStream(password string) js.Value {
	underlyingSource := map[string]interface{}{
		"start": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			controller := args[0]
			go func() {
				KeystoreBackupRaw(
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
		v := pair["value"].(string)
		backupLogger.Info("Loading " + k)

		err = idb.Set([]byte(k), []byte(v))
		if err != nil {
			return err
		}
		backupLogger.Info("OK")
	}

	return nil
}
