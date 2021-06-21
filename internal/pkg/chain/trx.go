package chain

import (
	"crypto/sha256"
	"time"

	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"

	guuid "github.com/google/uuid"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

//transaction will expire in 5 minutes (after send)
const Hours = 0
const Mins = 5
const Sec = 0

func CreateTrx(msgType quorumpb.TrxType, groupId string, data []byte) (*quorumpb.Trx, error) {
	var trx quorumpb.Trx

	trxId := guuid.New()
	trx.TrxId = trxId.String()
	trx.Type = msgType
	trx.GroupId = groupId
	trx.Sender = GetChainCtx().PeerId.Pretty()

	pubkey, err := getPubKey()
	if err != nil {
		return &trx, err
	}
	trx.Pubkey = pubkey
	trx.Data = data
	trx.TimeStamp = time.Now().UnixNano()
	trx.Version = GetChainCtx().Version
	timein := time.Now().Local().Add(time.Hour*time.Duration(Hours) +
		time.Minute*time.Duration(Mins) +
		time.Second*time.Duration(Sec))
	trx.Expired = timein.UnixNano()

	sign, err := signTrx(&trx)

	if err != nil {
		return &trx, err
	}

	trx.Signature = sign

	return &trx, nil
}

func Hash(data []byte) []byte {
	h := sha256.New()
	h.Write([]byte(data))
	hashed := h.Sum(nil)
	return hashed
}

func signTrx(trx *quorumpb.Trx) ([]byte, error) {
	bytes, err := proto.Marshal(trx)
	hashed := Hash(bytes)
	signature, err := GetChainCtx().Privatekey.Sign(hashed)
	return signature, err
}

func VerifyTrx(trx *quorumpb.Trx) (bool, error) {
	//clone trxMsg to verify
	clonetrxmsg := &quorumpb.Trx{
		TrxId:     trx.TrxId,
		Type:      trx.Type,
		GroupId:   trx.GroupId,
		Sender:    trx.Sender,
		Pubkey:    trx.Pubkey,
		Data:      trx.Data,
		TimeStamp: trx.TimeStamp,
		Version:   trx.Version,
		Expired:   trx.Expired}

	bytes, err := proto.Marshal(clonetrxmsg)
	if err != nil {
		return false, err
	}

	hashed := Hash(bytes)

	//create pubkey
	serializedpub, err := p2pcrypto.ConfigDecodeKey(trx.Pubkey)
	if err != nil {
		return false, err
	}

	pubkey, err := p2pcrypto.UnmarshalPublicKey(serializedpub)
	if err != nil {
		return false, err
	}

	verify, err := pubkey.Verify(hashed, trx.Signature)
	return verify, err
}
