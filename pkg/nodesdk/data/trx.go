package data

import (
	"encoding/hex"
	"time"

	guuid "github.com/google/uuid"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

const (
	Hours = 0
	Mins  = 0
	Sec   = 30
)

const OBJECT_SIZE_LIMIT = 200 * 1024 //(200Kb)

func CreateTrxWithoutSign(version string, nodesdkGroupItem *quorumpb.NodeSDKGroupItem, msgType quorumpb.TrxType, nonce int64, data []byte) (*quorumpb.Trx, []byte, error) {
	var trx quorumpb.Trx
	trxId := guuid.New()
	trx.TrxId = trxId.String()
	trx.Type = msgType
	trx.GroupId = nodesdkGroupItem.Group.GroupId
	trx.SenderPubkey = nodesdkGroupItem.Group.UserSignPubkey
	trx.Nonce = nonce

	var err error
	ciperKey, err := hex.DecodeString(nodesdkGroupItem.Group.CipherKey)
	if err != nil {
		return &trx, []byte(""), err
	}
	encryptdData, err := localcrypto.AesEncrypt(data, ciperKey)
	if err != nil {
		return &trx, []byte(""), err
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

func CreateTrx(version string, nodesdkGroupItem *quorumpb.NodeSDKGroupItem, msgType quorumpb.TrxType, nonce int64, data []byte) (*quorumpb.Trx, error) {
	trx, hashed, err := CreateTrxWithoutSign(version, nodesdkGroupItem, msgType, int64(nonce), data)
	if err != nil {
		return trx, err
	}

	// *huoju*
	// Signature can not be verified
	ks := localcrypto.GetKeystore()
	signature, err := ks.SignByKeyAlias(nodesdkGroupItem.SignAlias, hashed)
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

	hashed := localcrypto.Hash(bytes)

	//create pubkey
	serializedpub, err := p2pcrypto.ConfigDecodeKey(trx.SenderPubkey)
	if err != nil {
		return false, err
	}

	pubkey, err := p2pcrypto.UnmarshalPublicKey(serializedpub)
	if err != nil {
		return false, err
	}

	verify, err := pubkey.Verify(hashed, trx.SenderSign)
	return verify, err
}
