package data

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	guuid "github.com/google/uuid"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func CreateTrxWithoutSign(nodename, version, groupId, senderPubkey, CipherKey string, msgType quorumpb.TrxType, data []byte, encryptto ...[]string) (*quorumpb.Trx, error) {
	trx := &quorumpb.Trx{}

	trx.TrxId = guuid.New().String()
	trx.Type = msgType
	trx.GroupId = groupId
	trx.SenderPubkey = senderPubkey

	var encryptdData []byte

	var err error
	ciperKey, err := hex.DecodeString(CipherKey)
	if err != nil {
		return trx, err
	}
	encryptdData, err = localcrypto.AesEncrypt(data, ciperKey)
	if err != nil {
		return trx, err
	}

	trx.Data = encryptdData
	trx.Version = version
	trx.TimeStamp = time.Now().UnixNano()

	bytes, err := proto.Marshal(trx)
	if err != nil {
		return trx, err
	}
	hashed := localcrypto.Hash(bytes)
	trx.Hash = hashed

	return trx, nil
}

func CreateTrxByEthKey(nodename, version, groupId, senderPubkey, senderKeyname, CipherKey string, msgType quorumpb.TrxType, data []byte, encryptto ...[]string) (*quorumpb.Trx, error) {
	trx, err := CreateTrxWithoutSign(nodename, version, groupId, senderPubkey, CipherKey, msgType, data, encryptto...)
	if err != nil {
		return trx, err
	}

	ks := localcrypto.GetKeystore()
	var signature []byte

	signature, err = ks.EthSignByKeyName(senderKeyname, trx.Hash)
	if err != nil {
		return nil, err
	}

	trx.SenderSign = signature
	return trx, nil
}

func VerifyTrx(trx *quorumpb.Trx) (bool, error) {
	//clone trxMsg to verify

	trxClone := proto.Clone(trx).(*quorumpb.Trx)
	trxClone.Hash = nil
	trxClone.SenderSign = nil

	byts, err := proto.Marshal(trxClone)
	if err != nil {
		return false, err
	}
	hash := localcrypto.Hash(byts)
	if !bytes.Equal(hash, trx.Hash) {
		return false, nil
	}

	return VerifySign(trx.SenderPubkey, hash, trx.SenderSign)
}

func VerifySign(key string, hash, sign []byte) (bool, error) {
	//verify signature
	ks := localcrypto.GetKeystore()
	bytespubkey, err := base64.RawURLEncoding.DecodeString(key)
	if err == nil { //try eth key
		ethpubkey, err := ethcrypto.DecompressPubkey(bytespubkey)
		if err == nil {
			r := ks.EthVerifySign(hash, sign, ethpubkey)
			return r, nil
		}
		return false, err
	}
	return false, err
}
