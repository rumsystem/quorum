package data

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	guuid "github.com/google/uuid"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func CreateTrxWithoutSign(nodename string, version string, groupItem *quorumpb.GroupItemRumLite, msgType quorumpb.TrxType, data []byte) (*quorumpb.Trx, []byte, error) {
	var trxData []byte
	if groupItem.EncryptTrxCtn {
		ciperKey, err := hex.DecodeString(groupItem.CipherKey)
		if err != nil {
			return nil, nil, err
		}
		trxData, err = localcrypto.AesEncrypt(data, ciperKey)
		if err != nil {
			return nil, nil, err
		}
	} else {
		trxData = data
	}

	trx := &quorumpb.Trx{
		TrxId:        guuid.New().String(),
		Type:         msgType,
		GroupId:      groupItem.GroupId,
		SenderPubkey: groupItem.TrxSignPubkey,
		Version:      version,
		TimeStamp:    time.Now().UnixNano(),
		Data:         trxData,
	}

	bytes, err := proto.Marshal(trx)
	if err != nil {
		return nil, nil, err
	}

	hashed := localcrypto.Hash(bytes)

	return trx, hashed, nil
}

func CreateTrx(nodename string, version string, groupItem *quorumpb.GroupItemRumLite, msgType quorumpb.TrxType, data []byte) (*quorumpb.Trx, error) {
	trx, hash, err := CreateTrxWithoutSign(nodename, version, groupItem, msgType, data)
	if err != nil {
		return trx, err
	}

	//workaround for eth_sig, can we sign the hash by given pubkey
	ks := localcrypto.GetKeystore()

	//get keyname by using pubkey
	allKeys, err := ks.ListAll()
	if err != nil {
		return nil, err
	}

	keyname := ""
	for _, keyItem := range allKeys {
		pubkey, err := ks.GetEncodedPubkey(keyItem.Keyname, localcrypto.Sign)
		if err != nil {
			continue
		}
		if pubkey == groupItem.TrxSignPubkey {
			keyname = keyItem.Keyname
			break
		}
	}

	if keyname == "" {
		return nil, fmt.Errorf("keyname not found")
	}

	//sign it
	signature, err := ks.EthSignByKeyName(keyname, hash)
	if err != nil {
		return trx, err
	}

	trx.SenderSign = signature
	return trx, nil
}

func VerifyTrx(trx *quorumpb.Trx) (bool, error) {
	//clone trxMsg to verify
	clonetrxmsg := &quorumpb.Trx{
		TrxId:        trx.TrxId,
		Type:         trx.Type,
		GroupId:      trx.GroupId,
		SenderPubkey: trx.SenderPubkey,
		Data:         trx.Data,
		TimeStamp:    trx.TimeStamp,
		Version:      trx.Version,
	}

	bytes, err := proto.Marshal(clonetrxmsg)
	if err != nil {
		return false, err
	}
	hash := localcrypto.Hash(bytes)
	ks := localcrypto.GetKeystore()

	if len(trx.SenderPubkey) == 42 && trx.SenderPubkey[:2] == "0x" { //try 0x address
		//try verify 0x address
		sig := trx.SenderSign
		if sig[crypto.RecoveryIDOffset] == 27 || sig[crypto.RecoveryIDOffset] == 28 {
			sig[crypto.RecoveryIDOffset] -= 27
		}
		sigpubkey, err := ethcrypto.SigToPub(hash, sig)
		if err == nil {
			ok := ks.EthVerifySign(hash, trx.SenderSign, sigpubkey)
			if ok {
				addressfrompubkey := ethcrypto.PubkeyToAddress(*sigpubkey).Hex()
				if strings.EqualFold(addressfrompubkey, trx.SenderPubkey) {
					return true, nil
				} else {
					return false, fmt.Errorf("sig not match with the 0x address")
				}
			}
		}
	}

	bytespubkey, err := base64.RawURLEncoding.DecodeString(trx.SenderPubkey)
	if err == nil { //try eth key
		ethpubkey, err := ethcrypto.DecompressPubkey(bytespubkey)
		if err == nil {
			r := ks.EthVerifySign(hash, trx.SenderSign, ethpubkey)
			return r, nil
		}
		return false, err
	}

	return false, err
}
