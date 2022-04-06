package chain

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"

	guuid "github.com/google/uuid"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
)

const (
	Hours = 0
	Mins  = 0
	Sec   = 30
)

const OBJECT_SIZE_LIMIT = 200 * 1024 //(200Kb)

type TrxFactory struct {
	nodename  string
	groupId   string
	groupItem *quorumpb.GroupItem
}

func (factory *TrxFactory) Init(groupItem *quorumpb.GroupItem, nodename string) {
	factory.groupItem = groupItem
	factory.groupId = groupItem.GroupId
	factory.nodename = nodename
}

func (factory *TrxFactory) CreateTrxWithoutSign(msgType quorumpb.TrxType, data []byte, encryptto ...[]string) (*quorumpb.Trx, []byte, error) {
	var trx quorumpb.Trx

	trxId := guuid.New()
	trx.TrxId = trxId.String()
	trx.Type = msgType
	trx.GroupId = factory.groupItem.GroupId
	trx.SenderPubkey = factory.groupItem.UserSignPubkey
	nonce, err := nodectx.GetDbMgr().GetNextNouce(factory.groupId, factory.nodename)
	if err != nil {
		return &trx, []byte(""), err
	}

	trx.Nonce = int64(nonce)

	var encryptdData []byte
	if msgType == quorumpb.TrxType_POST && factory.groupItem.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
		//for post, private group, encrypted by age for all announced group users
		if len(encryptto) == 1 {
			var err error
			ks := localcrypto.GetKeystore()
			if len(encryptto[0]) == 0 {
				return &trx, []byte(""), fmt.Errorf("must have encrypt pubkeys for private group %s", factory.groupItem.GroupId)
			}
			encryptdData, err = ks.EncryptTo(encryptto[0], data)
			if err != nil {
				return &trx, []byte(""), err
			}

		} else {
			return &trx, []byte(""), fmt.Errorf("must have encrypt pubkeys for private group %s", factory.groupItem.GroupId)
		}

	} else {
		var err error
		ciperKey, err := hex.DecodeString(factory.groupItem.CipherKey)
		if err != nil {
			return &trx, []byte(""), err
		}
		encryptdData, err = localcrypto.AesEncrypt(data, ciperKey)
		if err != nil {
			return &trx, []byte(""), err
		}
	}

	trx.Data = encryptdData
	trx.Version = nodectx.GetNodeCtx().Version

	updateTrxTimeLimit(&trx)

	bytes, err := proto.Marshal(&trx)
	if err != nil {
		return &trx, []byte(""), err
	}
	hashed := localcrypto.Hash(bytes)
	return &trx, hashed, nil
}

// set TimeStamp and Expired for trx
func updateTrxTimeLimit(trx *quorumpb.Trx) {
	trx.TimeStamp = time.Now().UnixNano()
	timein := time.Now().Local().Add(time.Hour*time.Duration(Hours) +
		time.Minute*time.Duration(Mins) +
		time.Second*time.Duration(Sec))
	trx.Expired = timein.UnixNano()
}

func (factory *TrxFactory) CreateTrx(msgType quorumpb.TrxType, data []byte, encryptto ...[]string) (*quorumpb.Trx, error) {
	trx, hashed, err := factory.CreateTrxWithoutSign(msgType, data, encryptto...)
	if err != nil {
		return trx, err
	}
	ks := nodectx.GetNodeCtx().Keystore
	keyname := factory.groupItem.GroupId
	signature, err := ks.SignByKeyName(keyname, hashed)
	if err != nil {
		return trx, err
	}

	trx.SenderSign = signature

	return trx, nil
}

func (factory *TrxFactory) VerifyTrx(trx *quorumpb.Trx) (bool, error) {
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

func (factory *TrxFactory) GetUpdAppConfigTrx(item *quorumpb.AppConfigItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrx(quorumpb.TrxType_APP_CONFIG, encodedcontent)
}

func (factory *TrxFactory) GetChainConfigTrx(item *quorumpb.ChainConfigItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrx(quorumpb.TrxType_CHAIN_CONFIG, encodedcontent)
}

func (factory *TrxFactory) GetRegProducerTrx(item *quorumpb.ProducerItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}
	return factory.CreateTrx(quorumpb.TrxType_PRODUCER, encodedcontent)
}

func (factory *TrxFactory) GetRegUserTrx(item *quorumpb.UserItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}
	return factory.CreateTrx(quorumpb.TrxType_USER, encodedcontent)
}

func (factory *TrxFactory) GetAnnounceTrx(item *quorumpb.AnnounceItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrx(quorumpb.TrxType_ANNOUNCE, encodedcontent)
}

func (factory *TrxFactory) GetUpdSchemaTrx(item *quorumpb.SchemaItem) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrx(quorumpb.TrxType_SCHEMA, encodedcontent)
}

func (factory *TrxFactory) GetReqBlockRespTrx(requester string, block *quorumpb.Block, result quorumpb.ReqBlkResult) (*quorumpb.Trx, error) {
	var reqBlockRespItem quorumpb.ReqBlockResp
	reqBlockRespItem.Result = result
	reqBlockRespItem.ProviderPubkey = factory.groupItem.UserSignPubkey
	reqBlockRespItem.RequesterPubkey = requester
	reqBlockRespItem.GroupId = block.GroupId
	reqBlockRespItem.BlockId = block.BlockId

	pbBytesBlock, err := proto.Marshal(block)
	if err != nil {
		return nil, err
	}
	reqBlockRespItem.Block = pbBytesBlock

	bItemBytes, err := proto.Marshal(&reqBlockRespItem)
	if err != nil {
		return nil, err
	}

	//send ask next block trx out
	return factory.CreateTrx(quorumpb.TrxType_REQ_BLOCK_RESP, bItemBytes)
}

func (factory *TrxFactory) GetAskPeerIdTrx(req *quorumpb.AskPeerId) (*quorumpb.Trx, error) {
	bItemBytes, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrx(quorumpb.TrxType_ASK_PEERID, bItemBytes)
}

func (factory *TrxFactory) GetAskPeerIdRespTrx(req *quorumpb.AskPeerIdResp) (*quorumpb.Trx, error) {
	bItemBytes, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrx(quorumpb.TrxType_ASK_PEERID_RESP, bItemBytes)
}

func (factory *TrxFactory) GetReqBlockForwardTrx(block *quorumpb.Block) (*quorumpb.Trx, error) {
	var reqBlockItem quorumpb.ReqBlock
	reqBlockItem.BlockId = block.BlockId
	reqBlockItem.GroupId = block.GroupId
	reqBlockItem.UserId = factory.groupItem.UserSignPubkey

	bItemBytes, err := proto.Marshal(&reqBlockItem)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrx(quorumpb.TrxType_REQ_BLOCK_FORWARD, bItemBytes)
}

func (factory *TrxFactory) GetReqBlockBackwardTrx(block *quorumpb.Block) (*quorumpb.Trx, error) {
	var reqBlockItem quorumpb.ReqBlock
	reqBlockItem.BlockId = block.BlockId
	reqBlockItem.GroupId = block.GroupId
	reqBlockItem.UserId = factory.groupItem.UserSignPubkey

	bItemBytes, err := proto.Marshal(&reqBlockItem)
	if err != nil {
		return nil, err
	}

	return factory.CreateTrx(quorumpb.TrxType_REQ_BLOCK_BACKWARD, bItemBytes)
}

func (factory *TrxFactory) GetBlockProducedTrx(blk *quorumpb.Block) (*quorumpb.Trx, error) {
	encodedcontent, err := proto.Marshal(blk)
	if err != nil {
		return nil, err
	}
	return factory.CreateTrx(quorumpb.TrxType_BLOCK_PRODUCED, encodedcontent)
}

func (factory *TrxFactory) GetPostAnyTrx(content proto.Message, encryptto ...[]string) (*quorumpb.Trx, error) {
	encodedcontent, err := quorumpb.ContentToBytes(content)
	if err != nil {
		return nil, err
	}

	if binary.Size(encodedcontent) > OBJECT_SIZE_LIMIT {
		err := errors.New("Content size over 200Kb")
		return nil, err
	}

	return factory.CreateTrx(quorumpb.TrxType_POST, encodedcontent, encryptto...)
}
