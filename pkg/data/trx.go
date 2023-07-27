package data

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	guuid "github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	"github.com/rumsystem/quorum/pkg/pb"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

const DEFAULT_EPOCH_DURATION = 1000 //ms
const INIT_ANNOUNCE_TRX_ID = "00000000-0000-0000-0000-000000000000"
const INIT_FORK_TRX_ID = "00000000-0000-0000-0000-000000000001"

func CreateTrxWithoutSign(nodename string, version string, groupItem *quorumpb.GroupItem, msgType quorumpb.TrxType, data []byte, encryptto ...[]string) (*quorumpb.Trx, []byte, error) {
	var trx quorumpb.Trx

	trx.TrxId = guuid.New().String()
	trx.Type = msgType
	trx.GroupId = groupItem.GroupId
	trx.SenderPubkey = groupItem.UserSignPubkey

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
	trx.TimeStamp = time.Now().UnixNano()

	bytes, err := proto.Marshal(&trx)
	if err != nil {
		return &trx, []byte(""), err
	}
	hashed := localcrypto.Hash(bytes)
	return &trx, hashed, nil
}

func GetForkTrx(ks localcrypto.Keystore, item *quorumpb.GroupItem) (*quorumpb.Trx, *quorumpb.ConsensusInfo, error) {
	//create initial consensus for genesis block
	consensusInfo := &quorumpb.ConsensusInfo{
		ConsensusId:   uuid.New().String(),
		ChainVer:      0,
		InTrx:         INIT_FORK_TRX_ID,
		ForkFromBlock: 0,
	}

	//create fork info for genesis block
	forkItem := &quorumpb.ForkItem{
		GroupId:        item.GroupId,
		Consensus:      consensusInfo,
		StartFromBlock: 0,
		StartFromEpoch: 0,
		EpochDuration:  DEFAULT_EPOCH_DURATION,
		Producers:      []string{item.OwnerPubKey}, //owner is the first producer
		Memo:           "genesis fork",
	}

	//marshal fork item
	encodedcontent, err := proto.Marshal(forkItem)
	if err != nil {
		return nil, nil, err
	}

	ciperKey, err := hex.DecodeString(item.CipherKey)
	if err != nil {
		return nil, nil, err
	}
	encryptdData, err := localcrypto.AesEncrypt(encodedcontent, ciperKey)
	if err != nil {
		return nil, nil, err
	}

	trx := &quorumpb.Trx{
		TrxId:        INIT_FORK_TRX_ID,
		Type:         quorumpb.TrxType_FORK,
		GroupId:      item.GroupId,
		SenderPubkey: item.UserSignPubkey,
		Data:         encryptdData,
		TimeStamp:    time.Now().UnixNano(),
		Version:      nodectx.GetNodeCtx().Version,
	}

	//create hash
	byts, err := proto.Marshal(trx)
	if err != nil {
		return nil, nil, err
	}
	trx.Hash = localcrypto.Hash(byts)
	signature, err := ks.EthSignByKeyName(item.GroupId, trx.Hash)
	if err != nil {
		return nil, nil, err
	}
	trx.SenderSign = signature
	return trx, consensusInfo, nil
}

func GetAnnounceTrx(ks localcrypto.Keystore, item *pb.GroupItem) (*pb.Trx, error) {
	//owner announce as the first group producer
	aContent := &quorumpb.AnnounceContent{
		Type:          quorumpb.AnnounceType_AS_PRODUCER,
		SignPubkey:    item.OwnerPubKey,
		EncryptPubkey: item.UserEncryptPubkey,
		Memo:          "owner announce as the first group producer",
	}

	aItem := &quorumpb.AnnounceItem{
		GroupId:         item.GroupId,
		Action:          quorumpb.ActionType_ADD,
		Content:         aContent,
		AnnouncerPubkey: item.OwnerPubKey,
	}

	//create hash
	byts, err := proto.Marshal(aItem)
	if err != nil {
		return nil, err
	}
	aItem.Hash = localcrypto.Hash(byts)
	signature, err := ks.EthSignByKeyName(item.GroupId, aItem.Hash)
	if err != nil {
		return nil, err
	}

	aItem.Signature = signature

	//marshal fork item
	encodedcontent, err := proto.Marshal(aItem)
	if err != nil {
		return nil, err
	}

	//encrypt by cipher key
	ciperKey, err := hex.DecodeString(item.CipherKey)
	if err != nil {
		return nil, err
	}
	encryptdData, err := localcrypto.AesEncrypt(encodedcontent, ciperKey)
	if err != nil {
		return nil, err
	}

	trx := &quorumpb.Trx{
		TrxId:        INIT_ANNOUNCE_TRX_ID,
		Type:         quorumpb.TrxType_ANNOUNCE,
		GroupId:      item.GroupId,
		SenderPubkey: item.UserSignPubkey,
		Data:         encryptdData,
		TimeStamp:    time.Now().UnixNano(),
		Version:      nodectx.GetNodeCtx().Version,
	}

	//create hash
	byts, err = proto.Marshal(trx)
	if err != nil {
		return nil, err
	}
	trx.Hash = localcrypto.Hash(byts)
	signature, err = ks.EthSignByKeyName(item.GroupId, trx.Hash)
	if err != nil {
		return nil, err
	}
	trx.SenderSign = signature
	return trx, nil
}

func CreateTrxByEthKey(nodename string, version string, groupItem *quorumpb.GroupItem, msgType quorumpb.TrxType, data []byte, keyalias string, encryptto ...[]string) (*quorumpb.Trx, error) {
	trx, hash, err := CreateTrxWithoutSign(nodename, version, groupItem, msgType, data, encryptto...)
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
