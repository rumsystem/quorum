//go:build !js
// +build !js

package handlers

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"path/filepath"

	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/options"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"google.golang.org/protobuf/proto"
)

func CheckSignAndEncryptWithKeystore(keystoreName, keystoreDir, configDir, peerName, password string) error {
	ks, _, err := localcrypto.InitDirKeyStore(keystoreName, keystoreDir)
	if err != nil {
		return fmt.Errorf("localcrypto.InitKeystore failed: %s", err)
	}

	// get keysignmap from config
	nodeoptions, err := options.InitNodeOptions(configDir, peerName)
	if err != nil {
		return fmt.Errorf("options.InitNodeOptions failed: %s", err)
	}

	if err := ks.Unlock(nodeoptions.SignKeyMap, password); err != nil {
		return fmt.Errorf("ks.Unlock failed: %s", err)
	}

	for keyname, _ := range nodeoptions.SignKeyMap {
		// check signature
		if err := checkSignature(ks, keyname); err != nil {
			return err
		}

		// check encrypt
		{
			if keyname == "default" {
				continue
			}

			if err := checkEncrypt(ks, keyname, password); err != nil {
				return err
			}
		}
	}

	return nil
}

func getRandLength(a, b int) int {
	var res int
	for {
		res = rand.Intn(b)
		if res >= a {
			break
		}
	}

	return res
}

func checkSignature(ks *localcrypto.DirKeyStore, keyname string) error {
	length := getRandLength(10, 100)
	msg := utils.GetRandomStr(length)

	signature, err := ks.SignByKeyName(keyname, []byte(msg))
	if err != nil {
		return err
	}

	// should success
	if ok, err := ks.VerifySignByKeyName(keyname, []byte(msg), signature); err != nil || !ok {
		return errors.New("signature verify should success")
	}

	// should fail
	msg = utils.GetRandomStr(length)
	if ok, err := ks.VerifySignByKeyName(keyname, []byte(msg), signature); err != nil || ok {
		return errors.New("signature verify should fail")
	}
	return nil
}

func checkEncrypt(ks *localcrypto.DirKeyStore, keyname, password string) error {
	length := getRandLength(10, 100)
	data := utils.GetRandomStr(length)

	key, err := ks.LoadEncryptKey(localcrypto.Encrypt.NameString(keyname), password)
	if err != nil {
		return fmt.Errorf("ks.LoadEncryptKey failed: %s", err)
	}
	encryptid := key.Recipient().String()

	// should success
	encryptdata, err := ks.EncryptTo([]string{encryptid}, []byte(data))
	if err != nil {
		return nil
	}

	decrypteddata, err := ks.Decrypt(keyname, encryptdata)
	if err != nil {
		return err
	}

	if string(decrypteddata) != data {
		return fmt.Errorf("decrypt data is not matched with orginal: %s / %s", data, decrypteddata)
	}

	return nil
}

func loadAndDecryptTrx(blockDbDir, seedDir string) error {
	if !utils.DirExist(blockDbDir) {
		return fmt.Errorf("%s not exist", blockDbDir)
	}

	db := storage.QSBadger{}
	if err := db.Init(blockDbDir); err != nil {
		logger.Fatalf("init backuped db failed: %s", err)
	}
	defer db.Close()

	count := 0
	key := getBlockPrefixKey()
	err := db.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}

		if count > 10 { // just decrypt trx data in 10 blocks
			return nil
		}

		// decrypt trx data
		var blockChunk quorumpb.BlockDbChunk
		if err := proto.Unmarshal(v, &blockChunk); err != nil {
			return fmt.Errorf("proto.Unmarshal block data failed: %s", err)
		}
		block := blockChunk.BlockItem
		if block != nil {
			for _, trx := range block.Trxs {
				groupId := trx.GroupId
				if groupId == "" {
					groupId = block.GroupId
				}

				seedPath := filepath.Join(seedDir, fmt.Sprintf("%s.json", groupId))
				seed, err := loadGroupSeed(seedPath)
				if err != nil {
					logger.Warningf("load group seed from backuped file failed: %s", err)
					continue
				}
				ciperKey, err := hex.DecodeString(seed.CipherKey)
				if err != nil {
					return fmt.Errorf("get ciperKey failed: %s", err)
				}

				if _, err := localcrypto.AesDecode(trx.Data, ciperKey); err != nil {
					return fmt.Errorf("decrypt trx data failed: %s", err)
				}
			}
		}

		count++
		return nil
	})

	if err != nil {
		logger.Fatalf("dbManager.Db.PrefixForeach failed: %s", err)
	}
	return nil
}

func loadGroupSeed(path string) (*GroupSeed, error) {
	var seed GroupSeed
	seedByte, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(seedByte, &seed); err != nil {
		return nil, err
	}

	return &seed, nil
}
