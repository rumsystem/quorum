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
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

const (
	Hours = 0
	Mins  = 0
	Sec   = 30
)

const OBJECT_SIZE_LIMIT = 200 * 1024 //(200Kb)

func CreateTrxWithoutSign(nodename string, version string, groupItem *quorumpb.GroupItem, msgType quorumpb.TrxType, nonce int64, data []byte, encryptto ...[]string) (*quorumpb.Trx, []byte, error) {
	var trx quorumpb.Trx

	trxId := guuid.New()
	trx.TrxId = trxId.String()
	trx.Type = msgType
	trx.GroupId = groupItem.GroupId
	trx.SenderPubkey = groupItem.UserSignPubkey
	trx.Nonce = nonce

	var encryptdData []byte
	if msgType == quorumpb.TrxType_POST && groupItem.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
		//for post, private group, encrypted by age for all announced group users
		if len(encryptto) == 1 {
			var err error
			ks := localcrypto.GetKeystore()
			if len(encryptto[0]) == 0 {
				return &trx, []byte(""), fmt.Errorf("must have encrypt pubkeys for private group %s", groupItem.GroupId)
			}
			encryptdData, err = ks.EncryptTo(encryptto[0], data)
			if err != nil {
				return &trx, []byte(""), err
			}

		} else {
			return &trx, []byte(""), fmt.Errorf("must have encrypt pubkeys for private group %s", groupItem.GroupId)
		}

	} else {
		var err error
		ciperKey, err := hex.DecodeString(groupItem.CipherKey)
		if err != nil {
			return &trx, []byte(""), err
		}
		encryptdData, err = localcrypto.AesEncrypt(data, ciperKey)
		if err != nil {
			return &trx, []byte(""), err
		}
	}

	trx.Data = encryptdData
	trx.Version = version

	UpdateTrxTimeLimit(&trx)

	bytes, err := proto.Marshal(&trx)
	if err != nil {
		return &trx, []byte(""), err
	}
	hashed := localcrypto.Hash(bytes)
	return &trx, hashed, nil
}

func CreateTrxByEthKey(nodename string, version string, groupItem *quorumpb.GroupItem, msgType quorumpb.TrxType, nonce int64, data []byte, keyalias string, encryptto ...[]string) (*quorumpb.Trx, error) {
	trx, hash, err := CreateTrxWithoutSign(nodename, version, groupItem, msgType, int64(nonce), data, encryptto...)

	if err != nil {
		return trx, err
	}
	ks := localcrypto.GetKeystore()
	var signature []byte
	if keyalias == "" {
		keyname := groupItem.GroupId
		signature, err = ks.EthSignByKeyName(keyname, hash)
	} else {
		signature, err = ks.EthSignByKeyAlias(keyalias, hash)
	}

	if err != nil {
		return trx, err
	}
	trx.SenderSign = signature
	return trx, nil

}

// set TimeStamp and Expired for trx
func UpdateTrxTimeLimit(trx *quorumpb.Trx) {
	trx.TimeStamp = time.Now().UnixNano()
	timein := time.Now().Local().Add(time.Hour*time.Duration(Hours) +
		time.Minute*time.Duration(Mins) +
		time.Second*time.Duration(Sec))
	trx.Expired = timein.UnixNano()
}

func VerifyTrx(trx *quorumpb.Trx) (bool, error) {
	//clone trxMsg to verify
	clonetrxmsg := &quorumpb.Trx{
		TrxId:        trx.TrxId,
		Type:         trx.Type,
		GroupId:      trx.GroupId,
		SenderPubkey: trx.SenderPubkey,
		Nonce:        trx.Nonce,
		Data:         trx.Data,
		TimeStamp:    trx.TimeStamp,
		Version:      trx.Version,
		Expired:      trx.Expired}

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
			r := ks.EthVerifySign(hash, trx.SenderSign, sigpubkey)
			if r == true {
				addressfrompubkey := ethcrypto.PubkeyToAddress(*sigpubkey).Hex()
				if strings.ToLower(addressfrompubkey) == strings.ToLower(trx.SenderPubkey) {
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
	}
	//create pubkey
	serializedpub, err := p2pcrypto.ConfigDecodeKey(trx.SenderPubkey)
	if err != nil {
		return false, err
	}

	pubkey, err := p2pcrypto.UnmarshalPublicKey(serializedpub)
	if err != nil {
		pubkey, err = p2pcrypto.UnmarshalPublicKey(bytespubkey)
		if err != nil {
			return false, err
		}
	}

	verify, err := pubkey.Verify(hash, trx.SenderSign)
	return verify, err
}
